// Copyright © 2014 Lawrence E. Bakst. All rights reserved.

package hashtable

import (
	"fmt"
	"github.com/tildeleb/hashland/hashf"
	"github.com/tildeleb/cuckoo/primes"
)

type Bucket struct {
	Key []byte
}

type Stats struct {
	Inserts int
	Cols int
	Probes int
	Heads int
	Dups int
	Nbuckets int
	Entries int
	Q float64
	//
	Lines int
	Size uint64
	SizeLog2 uint64
	SizeMask uint64
}

type HashTable struct {
	Buckets [][]Bucket
	Stats
	extra int
	pd bool
	oa bool
	prime bool
}

// Henry Warren, "Hacker's Delight", ch. 5.3
func NextLog2(x uint32) uint32 {
	if x <= 1 {
		return x
	} 
	x--
	n := uint32(0)
	y := uint32(0)
	y = x >>16
	if y != 0 {
		n += 16
		x = y
	}
	y = x >> 8
	if y != 0 {
		n += 8
		x = y
	}
	y = x >> 4;
	if y != 0 {
		n +=  4
		x = y
	}
	y = x >> 2
	if y != 0 {
		n +=  2
		x = y
	}
	y = x >> 1
	if y != 0 {
		return n + 2
	}
	return n + x
}

func NewHashTable(size, extra int, pd, oa, prime bool) *HashTable {
	ht := new(HashTable)
	ht.Lines = size
	ht.SizeLog2 = uint64(NextLog2(uint32(ht.Lines)) + uint32(extra))
	ht.Size = 1 << ht.SizeLog2
	ht.prime = prime
	if prime {
		ht.Size = uint64(primes.NextPrime(int(ht.Size)))
	}
	ht.SizeMask = ht.Size - 1
	ht.Buckets = make([][]Bucket, ht.Size, ht.Size)
	return ht
}

func (ht *HashTable) Insert(ka []byte) {
	k := make([]byte, len(ka), len(ka))
	k = k[:]
	amt := copy(k, ka)
	if amt != len(ka) {
		panic("Add")
	}
	ht.Inserts++
	idx := uint64(0)
	h := hashf.Hashf(k) // jenkins.Hash232(k, 0)
	if ht.prime {
		idx = h % ht.Size
	} else {
		idx = h & ht.SizeMask
	}
	//fmt.Printf("index=%d\n", idx)
	cnt := 0
	pass := 0

	//fmt.Printf("Add: %x\n", k)
	//ht.Buckets[idx].Key = k
	//len(ht.Buckets[idx].Key) == 0
	for {
		if ht.Buckets[idx] == nil {
			// no entry or chain at this location, make it
			ht.Buckets[idx] = append(ht.Buckets[idx], Bucket{Key: k})
			//fmt.Printf("Add: idx=%d, len=%d, hash=0x%08x, key=%q\n", idx, len(ht.Buckets[idx]), h, ht.Buckets[idx][0].Key)
			ht.Probes++
			ht.Heads++
			return
		}
		if ht.oa {
			if cnt == 0 {
				ht.Cols++
			} else {
				ht.Probes++
			}

			// check for a duplicate key
			bh := hashf.Hashf(ht.Buckets[idx][0].Key)
			if bh == h {
				if ht.pd {
					fmt.Printf("hash=0x%08x, idx=%d, key=%q\n", h, idx, k)
					fmt.Printf("hash=0x%08x, idx=%d, key=%q\n", bh, idx, ht.Buckets[idx][0].Key)
				}
				ht.Dups++
			}
			idx++
			cnt++
			if idx > ht.Size - 1 {
				pass++
				if pass > 1 {
					panic("Add: pass")
				}
				idx = 0
			}
		} else {
			// first scan slice for dups
			for j := range ht.Buckets[idx] {
				bh := hashf.Hashf(ht.Buckets[idx][j].Key)
				//fmt.Printf("idx=%d, j=%d/%d, bh=0x%08x, h=0x%08x, key=%q\n", idx, j, len(ht.Buckets[idx]), bh, h, ht.Buckets[idx][j].Key)
				if bh == h {
					if ht.pd {
						fmt.Printf("idx=%d, j=%d/%d, bh=0x%08x, h=0x%08x, key=%q, bkey=%q\n", idx, j, len(ht.Buckets[idx]), bh, h, k, ht.Buckets[idx][j].Key)
						//fmt.Printf("hash=0x%08x, idx=%d, key=%q\n", h, idx, k)
						//fmt.Printf("hash=0x%08x, idx=%d, key=%q\n", bh, idx, ht.Buckets[idx][0].Key)
					}
					ht.Dups++
				}
			}
			// add element
			ht.Buckets[idx] = append(ht.Buckets[idx], Bucket{Key: k})
			ht.Probes++
			break
		}
	}
}

// The theoretical metric from "Red Dragon Book"
func (ht *HashTable) HashQuality() float64 {
	n := float64(0.0)
	buckets := 0
	entries := 0
	for _, v := range ht.Buckets {
		if v != nil {
			buckets++
			count := float64(len(v))
			entries += len(v)
			n += count * (count + 1.0)
		}
	}
	n *= float64(ht.Size)
	d := float64(ht.Inserts) * (float64(ht.Inserts) + 2.0 * float64(ht.Size) - 1.0) 	// (n / 2m) * (n + 2m - 1)
	//fmt.Printf("buckets=%d, entries=%d, inserts=%d, size=%d, n=%f, d=%f, n/d=%f\n", buckets, entries, ht.Inserts, ht.Size, n, d, n/d)
	ht.Nbuckets = buckets
	ht.Entries = entries
	ht.Q = n / d
	return n / d
}