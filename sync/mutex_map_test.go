package sync

import (
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMutexMap(t *testing.T) {
	a := assert.New(t)

	// To test that mutex map works correctly we will run a lot of goroutines concurrently and each goroutine will obtain the lock for a random key
	// and then output a sequence of strings "key 0", "key 1", ..., "key k". All the outputs are written to a single channel.
	// Since the number of goroutines will be much larger than the number of possible keys, there will be multiple goroutines that want the same lock at the same time.
	// The synchronization worked if the subsequence for each key in the output channel is sorted, i.e. of the form "key 0", ...., "key k", "key 0", ...., "key k", ... .
	mm := NewMutexMap()

	keyCount := 20
	// the length of the sequence produced by each goroutine
	outPerKey := 4
	waitTime := time.Microsecond
	//number of goroutines
	grCount := 10000
	output := make(chan string, grCount*outPerKey)

	wg := sync.WaitGroup{}
	wg.Add(grCount)
	for i := 0; i < grCount; i++ {
		go func() {
			defer wg.Done()
			key := randomKey(keyCount)
			l := mm.Lock(key)
			defer l.Unlock()
			for j := 0; j < outPerKey; j++ {
				output <- key + " " + strconv.Itoa(j)
				time.Sleep(waitTime)
			}
		}()
	}
	wg.Wait()
	close(output)

	a.Len(output, grCount*outPerKey)
	// MutexMap should be empty since no locks are held
	a.Empty(mm.keyToLock)

	// sort outputs by key
	outputByKey := make([][]int, keyCount)
	for o := range output {
		parts := strings.Split(o, " ")
		k, err := strconv.Atoi(parts[0])
		a.Nil(err)
		j, err := strconv.Atoi(parts[1])
		a.Nil(err)
		outputByKey[k] = append(outputByKey[k], j)
	}

	// check that outputs are in correct order, for each key we should get 0,1,2,3,...,outPerKey-1,0,1,2,3,...,outPerKey-1,...
	for i := 0; i < keyCount; i++ {
		//the length of the output for a given key should be a multiple of outPerKey
		a.Zero(len(outputByKey[i]) % outPerKey)
		for k := 0; k < len(outputByKey[i]); k += outPerKey {
			for j := 0; j < outPerKey; j++ {
				a.Equal(j, outputByKey[i][k+j])
			}
		}
	}
}

func randomKey(max int) string {
	k := rand.Intn(max)
	return strconv.Itoa(k)
}
