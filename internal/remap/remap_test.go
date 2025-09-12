package remap_test

import (
	"aegis/internal/remap"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

var indexRe = regexp.MustCompile("index.html")
var rootRe = regexp.MustCompile("^/$")
var imagesRe = regexp.MustCompile("^/images/.+")
var usersRe = regexp.MustCompile("^/users/$")

// TestRemapPut verifies the Put operation of ReMap by inserting multiple regex-value pairs
// and checking if all entries are correctly stored in the map.
func TestRemapPut(t *testing.T) {
	rm := remap.NewReMap[string]()
	rm.Put(indexRe, "index")
	rm.Put(rootRe, "root")
	rm.Put(imagesRe, "images")
	rm.Put(usersRe, "users")

	// Test all entries are added
	entries := rm.Entries()
	assert.Equal(t, entries, map[*regexp.Regexp]string{
		indexRe:  "index",
		rootRe:   "root",
		imagesRe: "images",
		usersRe:  "users",
	})
}

// TestRemapDelete validates the Delete operation of ReMap:
// 1. Deleting an existing entry should remove it from the map.
// 2. Deleting a non-existent entry should leave the map unchanged.
func TestRemapDelete(t *testing.T) {
	rm := remap.NewReMap[string]()
	rm.Put(indexRe, "index")
	rm.Put(rootRe, "root")
	rm.Put(imagesRe, "images")
	rm.Put(usersRe, "users")

	// Delete test for the existing entry
	rm.Delete(usersRe)
	entries := rm.Entries()
	assert.Equal(t, entries, map[*regexp.Regexp]string{
		indexRe:  "index",
		rootRe:   "root",
		imagesRe: "images",
	})

	// Delete test for a non-existent entry
	notExisting := regexp.MustCompile("UNDEFINED")
	rm.Delete(notExisting)
	assert.Equal(t, entries, map[*regexp.Regexp]string{
		indexRe:  "index",
		rootRe:   "root",
		imagesRe: "images",
	})
}

// TestRemapGet validates the Get operation of ReMap:
// 1. Retrieving an existing key should return the value and true.
// 2. Retrieving a non-existent key should return empty value and false.
func TestRemapGet(t *testing.T) {
	rm := remap.NewReMap[string]()
	rm.Put(indexRe, "index")
	rm.Put(rootRe, "root")
	rm.Put(imagesRe, "images")
	rm.Put(usersRe, "users")

	// Get existing entry
	it, found := rm.Get(rootRe)
	assert.True(t, found)
	assert.Equal(t, it, "root")

	// Get non-existent entry
	notExisting := regexp.MustCompile("UNDEFINED")
	it, found = rm.Get(notExisting)
	assert.False(t, found)
	assert.Empty(t, it)
}

// TestRemapFind verifies the Find operation of ReMap by testing path matching against regex patterns:
// 1. Non-matching path returns no results.
// 2. Exact match returns the corresponding value.
// 3. Nested path matches one pattern.
// 4. Overlapping patterns return multiple values in registration order.
func TestRemapFind(t *testing.T) {
	rm := remap.NewReMap[string]()
	rm.Put(indexRe, "index")
	rm.Put(rootRe, "root")
	rm.Put(imagesRe, "images")
	rm.Put(usersRe, "users")

	// Find undefined key
	vals, found := rm.Find("unknown")
	assert.False(t, found)
	assert.Empty(t, vals)

	// Find known key
	vals, found = rm.Find("/")
	assert.True(t, found)
	assert.Len(t, vals, 1)
	assert.Equal(t, vals[0], "root")

	// Find key "index" and check that there are no interactions with other entries
	vals, found = rm.Find("/users/index.html")
	assert.True(t, found)
	assert.Len(t, vals, 1)
	assert.Equal(t, vals[0], "index")

	// Find multiple entries by the patterns "/images" and "index.html"
	vals, found = rm.Find("/images/index.html")
	assert.True(t, found)
	assert.Len(t, vals, 2)
	assert.Equal(t, vals, []string{"index", "images"})
}
