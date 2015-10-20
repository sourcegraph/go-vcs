package internal

import (
	"path/filepath"
	"io/ioutil"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"time"
)

var gen rand.Source

func init() {
	gen = rand.NewSource(time.Now().UnixNano())
}

// Constructs platform-specific temporary script file with a given prefix
// On Windows such a file must have .bat extension
func ScriptFile(prefix string) (string, error) {
	
	if runtime.GOOS == "windows" {
		for {
        	tempFile := filepath.Join(os.TempDir(), prefix + strconv.FormatInt(gen.Int63(), 36) + ".bat")
        	_, err := os.Stat(tempFile)
        	if err != nil {
        		if os.IsNotExist(err) {
        			return filepath.ToSlash(tempFile), nil
        		} else {
        			return "", err
        		}
        	}
    	}
	} else {
		tf, err := ioutil.TempFile("", prefix)
		if err != nil {
			return "", err
		}
		tf.Close()
		return filepath.ToSlash(tf.Name()), nil
	}
}

// Wrapper around ioutil.WriteFile that updates permissions regardless if file existed before
func WriteFileWithPermissions(file string, content []byte, perm os.FileMode) error {
	fi, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	_,err = fi.Write(content)
	if err != nil {
		return err
	}
	fi.Close()
	return os.Chmod(file, perm)
}