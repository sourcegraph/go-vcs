.PHONY: benchmark.txt

benchmark.txt:
	go test -test.bench='.*' -test.benchmem > $@ 2>&1
	cat $@

bench-gogit:
	# XXX: Remove this before merging, just a bookmark.
	go test -v -bench=".*GitGoGit" -run XXX
