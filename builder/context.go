package builder

import (
  "bufio"
  "fmt"
  "io"
  "io/ioutil"
  "os"
  "os/exec"
  "path/filepath"
  "runtime"
  "strings"

  "github.com/docker/docker/pkg/archive"
  "github.com/docker/docker/pkg/fileutils"
  "github.com/docker/docker/pkg/gitutils"
  "github.com/docker/docker/pkg/httputils"
  "github.com/docker/docker/pkg/ioutils"
  "github.com/docker/docker/pkg/progress"
  "github.com/docker/docker/pkg/streamformatter"
)

// ValidateContextDirectory checks if all the contents of the directory
// can be read and returns an error if some files can't be read
// symlinks which point to non-existing files don't trigger an error
func ValidateContextDirectory(srcPath string, excludes []string) error {
  contextRoot, err := getContextRoot(srcPath)
  if err != nil {
    return err
  }
  return filepath.Walk(contextRoot, func(filePath string, f os.FileInfo, err error) error {
    if err != nil {
      if os.IsPermission(err) {
        return fmt.Errorf("can't stat '%s'", filePath)
      }
      if os.IsNotExist(err) {
        return nil
      }
      return err
    }

    // skip this directory/file if it's not in the path, it won't get added to the context
    if relFilePath, err := filepath.Rel(contextRoot, filePath); err != nil {
      return err
    } else if skip, err := fileutils.Matches(relFilePath, excludes); err != nil {
      return err
    } else if skip {
      if f.IsDir() {
        return filepath.SkipDir
      }
      return nil
    }

    // skip checking if symlinks point to non-existing files, such symlinks can be useful
    // also skip named pipes, because they hanging on open
    if f.Mode()&(os.ModeSymlink|os.ModeNamedPipe) != 0 {
      return nil
    }

    if !f.IsDir() {
      currentFile, err := os.Open(filePath)
      if err != nil && os.IsPermission(err) {
        return fmt.Errorf("no permission to read from '%s'", filePath)
      }
      currentFile.Close()
    }
    return nil
  })
}

// GetContextFromReader will read the contents of the given reader as either a
// Enginefile or tar archive. Returns a tar archive used as a context and a
// path to the Enginefile inside the tar.
func GetContextFromReader(r io.ReadCloser, enginefileName string) (out io.ReadCloser, relEnginefile string, err error) {
  buf := bufio.NewReader(r)

  magic, err := buf.Peek(archive.HeaderSize)
  if err != nil && err != io.EOF {
    return nil, "", fmt.Errorf("failed to peek context header from STDIN: %v", err)
  }

  if archive.IsArchive(magic) {
    return ioutils.NewReadCloserWrapper(buf, func() error { return r.Close() }), enginefileName, nil
  }

  // Input should be read as a Enginefile.
  tmpDir, err := ioutil.TempDir("", "providence-build-context-")
  if err != nil {
    return nil, "", fmt.Errorf("unbale to create temporary context directory: %v", err)
  }

  f, err := os.Create(filepath.Join(tmpDir, DefaultEnginefileName))
  if err != nil {
    return nil, "", err
  }
  _, err = io.Copy(f, buf)
  if err != nil {
    f.Close()
    return nil, "", err
  }

  if err := f.Close(); err != nil {
    return nil, "", err
  }
  if err := r.Close(); err != nil {
    return nil, "", err
  }

  tar, err := archive.Tar(tmpDir, archive.Uncompressed)
  if err != nil {
    return nil, "", err
  }

  return ioutils.NewReadCloserWrapper(tar, func() error {
    err := tar.Close()
    os.RemoveAll(tmpDir)
    return err
  }), DefaultEnginefileName, nil

}

// GetContextFromGitURL uses a Git URL as context for a `prov engine build`. The
// git repo is cloned into a temporary directory used as the context directory.
// Returns the absolute path to the temporary context directory, the relative
// path of the enginefile in that context directory, and a non-nil error on
// success.
func GetContextFromGitURL(gitURL, enginefileName string) (absContextDir, relEnginefile string, err error) {
  if _, err := exec.LookPath("git"); err != nil {
    return "", "", fmt.Errorf("unable to find 'git': %v", err)
  }
  if absContextDir, err = gitutils.Clone(gitURL); err != nil {
    return "", "", fmt.Errorf("unable to 'git clone' to temporary context directory: %v", err)
  }

  return getEnginefileRelPath(absContextDir, enginefileName)
}

// GetContextFromURL uses a remote URL as context for a `prov engine build`. The
// remote resource is downloaded as either a Enginefile or a tar archive.
// Returns the tar archive used for the context and a path of the
// enginefile inside the tar.
func GetContextFromURL(out io.Writer, remoteURL, enginefileName string) (io.ReadCloser, string, error) {
  response, err := httputils.Download(remoteURL)
  if err != nil {
    return nil, "", fmt.Errorf("unable to download remote context %s: %v", remoteURL, err)
  }
  progressOutput := streamformatter.NewStreamFormatter().NewProgressOutput(out, true)

  // Pass the response body through a progress reader.
  progReader := progress.NewProgressReader(response.Body, progressOutput, response.ContentLength, "", fmt.Sprintf("Downloading build context from remote url: %s", remoteURL))

  return GetContextFromReader(ioutils.NewReadCloserWrapper(progReader, func() error { return response.Body.Close() }), enginefileName)
}

// GetContextFromLocalDir uses the given local directory as context for a
// `prov engine build`. Returns the absolute path to the local context directory,
// the relative path of the enginefile in that context directory, and a non-nil
// error on success.
func GetContextFromLocalDir(localDir, enginefileName string) (absContextDir, relEnginefile string, err error) {
  // When using a local context directory, when the Enginefile is specified
  // with the `-f/--file` option then it is considered relative to the
  // current directory and not the context directory.
  if enginefileName != "" {
    if enginefileName, err = filepath.Abs(enginefileName); err != nil {
      return "", "", fmt.Errorf("unable to get absolute path to Enginefile: %v", err)
    }
  }

  return getEnginefileRelPath(localDir, enginefileName)
}

// getEnginefileRelPath uses the given context directory for a `prov engine build`
// and returns the absolute path to the context directory, the relative path of
// the enginefile in that context directory, and a non-nil error on success.
func getEnginefileRelPath(givenContextDir, givenEnginefile string) (absContextDir, relEnginefile string, err error) {
  if absContextDir, err = filepath.Abs(givenContextDir); err != nil {
    return "", "", fmt.Errorf("unable to get absolute context directory of given context directory %q: %v", givenContextDir, err)
  }

  // The context dir might be a symbolic link, so follow it to the actual
  // target directory.
  //
  // FIXME. We use isUNC (always false on non-Windows platforms) to workaround
  // an issue in golang. On Windows, EvalSymLinks does not work on UNC file
  // paths (those starting with \\). This hack means that when using links
  // on UNC paths, they will not be followed.
  if !isUNC(absContextDir) {
    absContextDir, err = filepath.EvalSymlinks(absContextDir)
    if err != nil {
      return "", "", fmt.Errorf("unable to evaluate symlinks in context path: %v", err)
    }
  }

  stat, err := os.Lstat(absContextDir)
  if err != nil {
    return "", "", fmt.Errorf("unable to stat context directory %q: %v", absContextDir, err)
  }

  if !stat.IsDir() {
    return "", "", fmt.Errorf("context must be a directory: %s", absContextDir)
  }

  absEnginefile := givenEnginefile
  if absEnginefile == "" {
    // No -f/--file was specified so use the default relative to the
    // context directory.
    absEnginefile = filepath.Join(absContextDir, DefaultEnginefileName)

    // Just to be nice ;-) look for 'enginefile' too but only
    // use it if we found it, otherwise ignore this check
    if _, err = os.Lstat(absEnginefile); os.IsNotExist(err) {
      altPath := filepath.Join(absContextDir, strings.ToLower(DefaultEnginefileName))
      if _, err = os.Lstat(altPath); err == nil {
        absEnginefile = altPath
      }
    }
  }

  // If not already an absolute path, the Enginefile path should be joined to
  // the base directory.
  if !filepath.IsAbs(absEnginefile) {
    absEnginefile = filepath.Join(absContextDir, absEnginefile)
  }

  // Evaluate symlinks in the path to the Enginefile too.
  //
  // FIXME. We use isUNC (always false on non-Windows platforms) to workaround
  // an issue in golang. On Windows, EvalSymLinks does not work on UNC file
  // paths (those starting with \\). This hack means that when using links
  // on UNC paths, they will not be followed.
  if !isUNC(absEnginefile) {
    absEnginefile, err = filepath.EvalSymlinks(absEnginefile)
    if err != nil {
      return "", "", fmt.Errorf("unable to evaluate symlinks in Enginefile path: %v", err)
    }
  }

  if _, err := os.Lstat(absEnginefile); err != nil {
    if os.IsNotExist(err) {
      return "", "", fmt.Errorf("Cannot locate Enginefile: %q", absEnginefile)
    }
    return "", "", fmt.Errorf("unable to stat Enginefile: %v", err)
  }

  if relEnginefile, err = filepath.Rel(absContextDir, absEnginefile); err != nil {
    return "", "", fmt.Errorf("unable to get relative Enginefile path: %v", err)
  }

  if strings.HasPrefix(relEnginefile, ".."+string(filepath.Separator)) {
    return "", "", fmt.Errorf("The Enginefile (%s) must be within the build context (%s)", givenEnginefile, givenContextDir)
  }

  return absContextDir, relEnginefile, nil
}

// isUNC returns true if the path is UNC (one starting \\). It always returns
// false on Linux.
func isUNC(path string) bool {
  return runtime.GOOS == "windows" && strings.HasPrefix(path, `\\`)
}
