build:
	go build -o ./bin/gogoevents .

test:
	go test -v -count=1 ./...

benchmark:
	go test -bench ./...

clean:
	rm -r ./bin/*