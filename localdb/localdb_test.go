package localdb

import (
	"fmt"
	"testing"

	"github.com/dgraph-io/badger/v4"
)

func TestBasic(t *testing.T) {
	err := Open()
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer Close()

	key := []byte("testKey")
	value := "testValue"

	// Store the key/value
	err = Set(key, []byte(value))
	if err != nil {
		t.Fatalf("Failed to store data: %v", err)
	}

	// Retrieve the key/value
	retrievedValue, err := Get(key)
	if err != nil {
		t.Fatalf("Failed to retrieve data: %v", err)
	}

	if string(retrievedValue) != value {
		t.Errorf("Expected value '%v', got '%v'", value, retrievedValue)
	}
}

const TOTAL_TEST_KEYS = 980

func TestGetKeysWithPrefix(t *testing.T) {
	err := Open()
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer Close()
	for i := range TOTAL_TEST_KEYS {
		key := fmt.Appendf([]byte(""), "testKey:%05d", i)
		value := fmt.Appendf([]byte(""), "testValue:%05d", i)
		// Store the key/value
		err := Set(key, []byte(value))
		if err != nil {
			t.Fatalf("Failed to store data: %v", err)
		}
	}

	prefix := []byte("testKey:")
	next := prefix
	count := 0
	for {
		keys, n, err := GetKeysWithPrefix(prefix, next, 10)
		if err != nil {
			t.Fatalf("Failed to retrieve keys: %v", err)
		}

		//t.Log("Next key", string(n))
		for _, _ = range keys {
			//t.Log(string(k))
			count += 1
		}
		if n == nil {
			break
		}
		next = n
	}
	if count != TOTAL_TEST_KEYS {
		t.Errorf("Expected %d keys, got %d", TOTAL_TEST_KEYS, count)
	}
	t.Logf("Inserted %d, feteched %d keys\n", count, count)
}

func TestCleanup(t *testing.T) {
	// Delete test keys
	err := Open()
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer Close()

	opts := badger.DefaultIteratorOptions
	opts.Prefix = []byte("testKey")

	count := 0
	err = db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			err := item.Value(func(v []byte) error {
				//t.Logf("Deleting Key: %s\n", item.Key())
				Delete(item.Key())
				count++
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	t.Logf("Deleted %d keys", count)
	if err != nil {
		t.Fatalf("Failed to delete keys: %v", err)
	}

}
