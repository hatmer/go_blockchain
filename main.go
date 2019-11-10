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

const BLOCK_SIZE = 4 //1024
const BUFFER_SIZE = 4096
const PORT = 8080
const DATA_FILE = "blockchain.dat"
const DIFFICULTY = 2

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
	f, err := os.OpenFile(DATA_FILE, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
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
	target := strings.Repeat("0", DIFFICULTY)
	for found := false; !found; {
		trial = string(sha([]byte(string(num) + x)))
		num += 1

		if string([]rune(trial)[0:DIFFICULTY]) == target {
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
 * put_index = 0
 * get_index = 0
 *
 * len_this_round = block_size
 * while len_this_round > 1:
 *   while put_index < (len_this_round / 2):
 *     slice[put_index] = sha(slice[get_index] + slice[get_index+1])
 *     put_index += 1
 *     get_index += 2
 *   len_this_round /= 2
 *
 */

func getMerkleRoot(entries []entry) (string, error) {
	if len(entries) != BLOCK_SIZE {
		return "", errors.New("incorrect block size\n")
	}

	// hash each entry
	var hashes [][]byte
	for _, x := range entries {
		hash := sha([]byte(x))
		hashes = append(hashes, hash)
	}

	// calculate Merkle Root in place
	get_index := 0
	for len_this_round := BLOCK_SIZE; len_this_round > 1; len_this_round /= 2 {
		get_index = 0
		for put_index := 0; put_index < (len_this_round / 2); put_index += 1 {
			to_hash := append([]byte(entries[get_index]), []byte(entries[get_index+1])...)
			hashes[put_index] = sha(to_hash)
			get_index += 2
			fmt.Printf("%d\n", get_index)
		}
	}
	return string(hashes[0]), nil
}

/*
 * generates block
 */
func generate(ch chan entry, hashPrev string) {
	entries := make([]entry, 0, BLOCK_SIZE)
	log.Printf("Starting new block. Block number: %d\n", blockNumber)
	for i := 0; i < BLOCK_SIZE; i += 1 {
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
	/*f, err := os.Create(DATA_FILE)
	if err != nil {
		panic(err)
	}
	f.Close()*/

	/* run block builder goroutine */
	ch := make(chan entry, BUFFER_SIZE)
	go generate(ch, seed)

	/* run server */
	p := dataPasser{ch}
	http.HandleFunc("/", p.handler)
	port := ":" + strconv.Itoa(PORT)
	log.Fatal(http.ListenAndServe(port, nil))
}
