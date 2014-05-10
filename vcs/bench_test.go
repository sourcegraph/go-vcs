package vcs

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"
)

const (
	benchFileSystemCommits = 15
	benchGetCommitCommits  = 15
	benchCommitLogCommits  = 15
)

func BenchmarkFileSystem_GitNative(b *testing.B) {
	defer func() {
		b.StopTimer()
		removeTmpDirs()
		b.StartTimer()
	}()

	cmds, files := makeGitCommandsAndFiles(benchFileSystemCommits)
	repo := makeGitRepositoryNative(b, cmds...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchFileSystem(b, repo, "mytag", files)
	}
}

func BenchmarkFileSystem_GitLibGit2(b *testing.B) {
	defer func() {
		b.StopTimer()
		removeTmpDirs()
		b.StartTimer()
	}()

	cmds, files := makeGitCommandsAndFiles(benchFileSystemCommits)
	repo := makeGitRepositoryLibGit2(b, cmds...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchFileSystem(b, repo, "mytag", files)
	}
}

func BenchmarkFileSystem_GitCmd(b *testing.B) {
	defer func() {
		b.StopTimer()
		removeTmpDirs()
		b.StartTimer()
	}()

	cmds, files := makeGitCommandsAndFiles(benchFileSystemCommits)
	repo := &GitRepositoryCmd{initGitRepository(b, cmds...)}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchFileSystem(b, repo, "mytag", files)
	}
}

func BenchmarkFileSystem_HgNative(b *testing.B) {
	defer func() {
		b.StopTimer()
		removeTmpDirs()
		b.StartTimer()
	}()

	cmds, files := makeHgCommandsAndFiles(benchFileSystemCommits)
	repo := makeHgRepositoryNative(b, cmds...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchFileSystem(b, repo, "mytag", files)
	}
}

func BenchmarkFileSystem_HgCmd(b *testing.B) {
	defer func() {
		b.StopTimer()
		removeTmpDirs()
		b.StartTimer()
	}()

	cmds, files := makeHgCommandsAndFiles(benchFileSystemCommits)
	repo := &HgRepositoryCmd{initHgRepository(b, cmds...)}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchFileSystem(b, repo, "mytag", files)
	}
}

func BenchmarkGetCommit_GitNative(b *testing.B) {
	defer func() {
		b.StopTimer()
		removeTmpDirs()
		b.StartTimer()
	}()

	cmds, _ := makeGitCommandsAndFiles(benchGetCommitCommits)
	repo := makeGitRepositoryNative(b, cmds...)
	openRepo := func() benchRepository {
		r, err := OpenGitRepositoryNative(repo.(*GitRepositoryNative).dir)
		if err != nil {
			b.Fatal(err)
		}
		return r
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchGetCommit(b, openRepo, "mytag")
	}
}

func BenchmarkGetCommit_GitLibGit2(b *testing.B) {
	defer func() {
		b.StopTimer()
		removeTmpDirs()
		b.StartTimer()
	}()

	cmds, _ := makeGitCommandsAndFiles(benchGetCommitCommits)
	repo := makeGitRepositoryLibGit2(b, cmds...)
	openRepo := func() benchRepository {
		r, err := OpenGitRepositoryLibGit2(repo.dir)
		if err != nil {
			b.Fatal(err)
		}
		return r
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchGetCommit(b, openRepo, "mytag")
	}
}

func BenchmarkGetCommit_GitCmd(b *testing.B) {
	defer func() {
		b.StopTimer()
		removeTmpDirs()
		b.StartTimer()
	}()

	cmds, _ := makeGitCommandsAndFiles(benchGetCommitCommits)
	openRepo := func() benchRepository { return &GitRepositoryCmd{initGitRepository(b, cmds...)} }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchGetCommit(b, openRepo, "mytag")
	}
}

func BenchmarkGetCommit_HgNative(b *testing.B) {
	defer func() {
		b.StopTimer()
		removeTmpDirs()
		b.StartTimer()
	}()

	cmds, _ := makeHgCommandsAndFiles(benchGetCommitCommits)
	repo := makeHgRepositoryNative(b, cmds...)
	openRepo := func() benchRepository {
		r, err := OpenHgRepositoryNative(repo.dir)
		if err != nil {
			b.Fatal(err)
		}
		return r
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchGetCommit(b, openRepo, "mytag")
	}
}

func BenchmarkGetCommit_HgCmd(b *testing.B) {
	defer func() {
		b.StopTimer()
		removeTmpDirs()
		b.StartTimer()
	}()

	cmds, _ := makeHgCommandsAndFiles(benchGetCommitCommits)
	openRepo := func() benchRepository { return &HgRepositoryCmd{initHgRepository(b, cmds...)} }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchGetCommit(b, openRepo, "mytag")
	}
}

func BenchmarkCommitLog_GitNative(b *testing.B) {
	defer func() {
		b.StopTimer()
		removeTmpDirs()
		b.StartTimer()
	}()

	cmds, _ := makeGitCommandsAndFiles(benchCommitLogCommits)
	repo := makeGitRepositoryNative(b, cmds...)
	openRepo := func() benchRepository {
		r, err := OpenGitRepositoryNative(repo.(*GitRepositoryNative).dir)
		if err != nil {
			b.Fatal(err)
		}
		return r
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchCommitLog(b, openRepo, "mytag")
	}
}

func BenchmarkCommitLog_GitLibGit2(b *testing.B) {
	defer func() {
		b.StopTimer()
		removeTmpDirs()
		b.StartTimer()
	}()

	cmds, _ := makeGitCommandsAndFiles(benchCommitLogCommits)
	repo := makeGitRepositoryLibGit2(b, cmds...)
	openRepo := func() benchRepository {
		r, err := OpenGitRepositoryLibGit2(repo.dir)
		if err != nil {
			b.Fatal(err)
		}
		return r
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchCommitLog(b, openRepo, "mytag")
	}
}

func BenchmarkCommitLog_GitCmd(b *testing.B) {
	defer func() {
		b.StopTimer()
		removeTmpDirs()
		b.StartTimer()
	}()

	cmds, _ := makeGitCommandsAndFiles(benchCommitLogCommits)
	openRepo := func() benchRepository { return &GitRepositoryCmd{initGitRepository(b, cmds...)} }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchCommitLog(b, openRepo, "mytag")
	}
}

func BenchmarkCommitLog_HgNative(b *testing.B) {
	defer func() {
		b.StopTimer()
		removeTmpDirs()
		b.StartTimer()
	}()

	cmds, _ := makeHgCommandsAndFiles(benchCommitLogCommits)
	repo := makeHgRepositoryNative(b, cmds...)
	openRepo := func() benchRepository {
		r, err := OpenHgRepositoryNative(repo.dir)
		if err != nil {
			b.Fatal(err)
		}
		return r
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchCommitLog(b, openRepo, "mytag")
	}
}

func BenchmarkCommitLog_HgCmd(b *testing.B) {
	defer func() {
		b.StopTimer()
		removeTmpDirs()
		b.StartTimer()
	}()

	cmds, _ := makeHgCommandsAndFiles(benchCommitLogCommits)
	openRepo := func() benchRepository { return &HgRepositoryCmd{initHgRepository(b, cmds...)} }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchCommitLog(b, openRepo, "mytag")
	}
}

func makeGitCommandsAndFiles(n int) (cmds, files []string) {
	for i := 0; i < n; i++ {
		name := benchFilename(i)
		files = append(files, name)
		cmds = append(cmds,
			fmt.Sprintf("mkdir -p %s", filepath.Dir(name)),
			fmt.Sprintf("echo hello%d >> %s", i, name),
			fmt.Sprintf("git add %s", name),
			fmt.Sprintf("GIT_COMMITTER_NAME=a GIT_COMMITTER_EMAIL=a@a.com GIT_COMMITTER_DATE=2014-05-06T19:20:21Z git commit -m commit%d --author='a <a@a.com>' --date 2014-05-06T19:20:21Z", i),
		)
	}
	cmds = append(cmds, "git tag mytag")
	return cmds, files
}

func makeHgCommandsAndFiles(n int) (cmds []string, files []string) {
	for i := 0; i < n; i++ {
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

type benchRepository interface {
	ResolveRevision(string) (CommitID, error)
	ResolveTag(string) (CommitID, error)
	GetCommit(CommitID) (*Commit, error)
	CommitLog(CommitID) ([]*Commit, error)
	FileSystem(CommitID) (FileSystem, error)
}

func benchFileSystem(b *testing.B, repo benchRepository, tag string, files []string) {
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

func benchGetCommit(b *testing.B, openRepo func() benchRepository, tag string) {
	repo := openRepo()

	commitID, err := repo.ResolveTag(tag)
	if err != nil {
		b.Errorf("ResolveTag: %s", err)
		return
	}

	_, err = repo.GetCommit(commitID)
	if err != nil {
		b.Errorf("GetCommit: %s", err)
		return
	}
}

func benchCommitLog(b *testing.B, openRepo func() benchRepository, tag string) {
	repo := openRepo()

	commitID, err := repo.ResolveTag(tag)
	if err != nil {
		b.Errorf("ResolveTag: %s", err)
		return
	}

	_, err = repo.CommitLog(commitID)
	if err != nil {
		b.Errorf("CommitLog: %s", err)
		return
	}
}
