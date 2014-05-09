package vcs

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"
)

const benchmarkCommits = 15

func BenchmarkGit(b *testing.B) {
	defer func() {
		b.StopTimer()
		removeTmpDirs()
		b.StartTimer()
	}()

	var cmds, files []string
	for i := 0; i < benchmarkCommits; i++ {
		name := benchFilename(i)
		files = append(files, name)
		cmds = append(cmds,
			fmt.Sprintf("mkdir -p %s", filepath.Dir(name)),
			fmt.Sprintf("echo hello%d >> %s", i, name),
			fmt.Sprintf("git add %s", name),
			fmt.Sprintf("GIT_COMMITTER_DATE=2014-05-06T19:20:21Z git commit -m commit%d --author='a <a@a.com>' --date 2014-05-06T19:20:21Z", i),
		)
	}
	cmds = append(cmds, "git tag mytag")
	repo := makeLocalGitRepository(b, cmds...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bench(b, repo, "mytag", files)
	}
}

func BenchmarkHg(b *testing.B) {
	defer func() {
		b.StopTimer()
		removeTmpDirs()
		b.StartTimer()
	}()

	cmds, files := makeHgCommandsAndFiles()
	repo := makeLocalHgRepository(b, false, cmds...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bench(b, repo, "mytag", files)
	}
}

func BenchmarkPythonHg(b *testing.B) {
	defer func() {
		b.StopTimer()
		removeTmpDirs()
		b.StartTimer()
	}()

	cmds, files := makeHgCommandsAndFiles()
	repo := makeLocalHgRepository(b, true, cmds...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bench(b, repo, "mytag", files)
	}
}

func makeHgCommandsAndFiles() (cmds []string, files []string) {
	for i := 0; i < benchmarkCommits; i++ {
		name := benchFilename(i)
		files = append(files, name)
		cmds = append(cmds,
			fmt.Sprintf("mkdir -p %s", filepath.Dir(name)),
			fmt.Sprintf("echo hello%d >> %s", i, name),
			fmt.Sprintf("hg add %s", name),
			fmt.Sprintf("hg commit -m hello%d --user 'a <a@a.com>' --date '2014-05-06 19:20:21 UTC'", i),
		)
	}
	cmds = append(cmds, "hg tag mytag")
	return cmds, files
}

func benchFilename(i int) string {
	switch i % 4 {
	case 0:
		return fmt.Sprintf("file%d", i)
	case 1:
		return fmt.Sprintf("dir%d/file%d", i%10, i)
	case 2:
		return fmt.Sprintf("dir%d/subdir%d/file%d", i%7, i%3, i)
	case 3:
		return fmt.Sprintf("file%d", i%2)
	}
	panic("unreachable")
}

func bench(b *testing.B, repo Repository, tag string, files []string) {
	commitID, err := repo.ResolveTag(tag)
	if err != nil {
		b.Errorf("ResolveTag: %s", err)
		return
	}

	fs, err := repo.FileSystem(commitID)
	if err != nil {
		b.Errorf("FileSystem: %s", err)
		return
	}

	for _, f := range files {
		dir := filepath.Dir(f)

		if dir != "." {
			// dir should exist and be a dir.
			dir1Info, err := fs.Stat(dir)
			if err != nil {
				b.Errorf("fs.Stat(%q): %s", dir, err)
				return
			}
			if !dir1Info.Mode().IsDir() {
				b.Errorf("dir %q stat !IsDir", dir)
			}

			// dir should contain an entry file1.
			dirEntries, err := fs.ReadDir(dir)
			if err != nil {
				b.Errorf("fs.ReadDir(dir): %s", err)
				return
			}
			if len(dirEntries) == 0 {
				b.Errorf("dir should contain file1")
				return
			}
		}

		// file should exist, and be a file.
		file, err := fs.Open(f)
		if err != nil {
			b.Errorf("fs.Open(%q): %s", f, err)
			return
		}
		_, err = ioutil.ReadAll(file)
		if err != nil {
			b.Errorf("ReadAll(%q): %s", f, err)
			return
		}
		file.Close()

		fi, err := fs.Stat(f)
		if err != nil {
			b.Errorf("fs.Stat(%q): %s", f, err)
			return
		}
		if !fi.Mode().IsRegular() {
			b.Errorf("file %q stat !IsRegular", f)
		}
	}
}
