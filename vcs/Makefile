.PHONY: benchmark.txt

benchmark.txt:
	go test -test.bench='.*' -test.benchmem > benchmark.txt 2>&1
