package main

import (
	"bufio"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const blockSize = 4 //1024
const bufferSize = 4096
const port = 8080
const dataFile = "blockchain.dat"
const difficulty = 2

var blockNumber = 0

type entry string

type block struct {
	blockHash      string
	hashPrevBlock  string
	hashMerkleRoot string
	time           int64
	entries        []entry
}

type dataPasser struct {
	ch chan entry
}

/*
 * write block to disk
 */
func (b *block) write() error {
	/* open output file */
	f, err := os.OpenFile(dataFile, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return errors.New("could not open output file\n")
	}
	defer func() {
		if err := f.Close(); err != nil {
			panic(err)
		}
	}()

	w := bufio.NewWriter(f)
	output := fmt.Sprintf("%x\n", b.blockHash)
	output += fmt.Sprintf("%x\n", b.hashPrevBlock)
	output += fmt.Sprintf("%x\n", b.hashMerkleRoot)
	output += fmt.Sprint(b.time) + "\n"
	for _, e := range b.entries {
		output += string(e) + "\n"
	}
	output += "\n"

	if _, err := w.WriteString(output); err != nil {
		panic(err)
	}

	if err = w.Flush(); err != nil {
		panic(err)
	}
	log.Print("wrote block to disk\n")
	return nil
}

/*
 * double SHA256
 */
func sha(x []byte) []byte {
	once := sha256.Sum256(x)
	twice := sha256.Sum256(once[:])
	return twice[:]
}

/*
 * PoW generator
 */
func pow(x string) (string, error) {
	trial := ""
	num := 0
	target := strings.Repeat("0", difficulty)
	for found := false; !found; {
		trial = string(sha([]byte(string(num) + x)))
		num += 1

		if string([]rune(trial)[0:difficulty]) == target {
			found = true
			log.Printf("number of hash iterations: %d\n", num)
			log.Printf("found block hash: %x\n", string(trial))
		}
	}
	return trial, nil

}

/*
 * calculates merkle root
 */

/*
 * Algorithm:
 *
 * putIndex = 0
 * getIndex = 0
 *
 * lenThisRound = blockSize
 * while lenThisRound > 1:
 *   while putIndex < (lenThisRound / 2):
 *     slice[putIndex] = sha(slice[getIndex] + slice[getIndex+1])
 *     putIndex += 1
 *     getIndex += 2
 *   lenThisRound /= 2
 *
 */

func getMerkleRoot(entries []entry) (string, error) {
	if len(entries) != blockSize {
		return "", errors.New("incorrect block size\n")
	}

	// hash each entry
	var hashes [][]byte
	for _, x := range entries {
		hash := sha([]byte(x))
		hashes = append(hashes, hash)
	}

	// calculate Merkle Root in place
	getIndex := 0
	for lenThisRound := blockSize; lenThisRound > 1; lenThisRound /= 2 {
		getIndex = 0
		for putIndex := 0; putIndex < (lenThisRound / 2); putIndex += 1 {
			toHash := append([]byte(entries[getIndex]), []byte(entries[getIndex+1])...)
			hashes[putIndex] = sha(toHash)
			getIndex += 2
			fmt.Printf("%d\n", getIndex)
		}
	}
	return string(hashes[0]), nil
}

/*
 * generates block
 */
func generate(ch chan entry, hashPrev string) {
	entries := make([]entry, 0, blockSize)
	log.Printf("Starting new block. Block number: %d\n", blockNumber)
	for i := 0; i < blockSize; i += 1 {
		val := <-ch
		entries = append(entries, val)
	}
	t := time.Now().Unix()
	merkleRoot, err := getMerkleRoot(entries[:])
	if err == nil {
		blockHash, err := pow(hashPrev + merkleRoot + string(t))
		b := block{blockHash, hashPrev, merkleRoot, t, entries}

		writeErr := (&b).write()
		if writeErr != nil {
			panic(err)
		}

		blockNumber += 1
		generate(ch, blockHash)
	} else {
		log.Printf("could not generate block. crashing...\n")
		os.Exit(1)
	}
}

/*
 * handles http request, writes to channel, and writes response (status and block number)
 */
func (p *dataPasser) handler(w http.ResponseWriter, r *http.Request) {
	input := entry(r.URL.Path[1:])
	log.Printf("input: %s\n", input)
	p.ch <- input
	fmt.Fprintf(w, "Block Number is %d\n", blockNumber)
}

/*
 * runs server on localhost 8080
 */
func main() {
	seed := string(sha([]byte("seed")))

	/* create output file */
	/*f, err := os.Create(dataFile)
	if err != nil {
		panic(err)
	}
	f.Close()*/

	/* run block builder goroutine */
	ch := make(chan entry, bufferSize)
	go generate(ch, seed)

	/* run server */
	p := dataPasser{ch}
	http.HandleFunc("/", p.handler)
	port := ":" + strconv.Itoa(port)
	log.Fatal(http.ListenAndServe(port, nil))
}
