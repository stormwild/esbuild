package cache

import (
	"sync"

	"github.com/evanw/esbuild/internal/logger"
	"github.com/evanw/esbuild/internal/runtime"
)

// This is a cache of the parsed contents of a set of files. The idea is to be
// able to reuse the results of parsing between builds and make subsequent
// builds faster by avoiding redundant parsing work. This only works if:
//
// * The AST information in the cache must be considered immutable. There is
//   no way to enforce this in Go, but please be disciplined about this. The
//   ASTs are shared in between builds. Any information that must be mutated
//   in the AST during a build must be done on a shallow clone of the data if
//   the mutation happens after parsing (i.e. a clone that clones everything
//   that will be mutated and shares only the parts that won't be mutated).
//
// * The information in the cache must not depend at all on the contents of
//   any file other than the file being cached. Invalidating an entry in the
//   cache does not also invalidate any entries that depend on that file, so
//   caching information that depends on other files can result in incorrect
//   results due to reusing stale data. For example, do not "bake in" some
//   value imported from another file.
//
// * Cached ASTs must only be reused if the parsing options are identical
//   between builds. For example, it would be bad if the AST parser depended
//   on options inherited from a nearby "package.json" file but those options
//   were not part of the cache key. Then the cached AST could incorrectly be
//   reused even if the contents of that "package.json" file have changed.
//
type CacheSet struct {
	SourceIndexCache *SourceIndexCache
	FSCache          *FSCache
	CSSCache         *CSSCache
	JSONCache        *JSONCache
	JSCache          *JSCache
}

func MakeCacheSet() CacheSet {
	return CacheSet{
		SourceIndexCache: &SourceIndexCache{
			entries:         make(map[logger.Path]uint32),
			nextSourceIndex: runtime.SourceIndex + 1,
		},
		FSCache: &FSCache{
			entries: make(map[string]*fsEntry),
		},
		CSSCache: &CSSCache{
			entries: make(map[logger.Path]*cssCacheEntry),
		},
		JSONCache: &JSONCache{
			entries: make(map[logger.Path]*jsonCacheEntry),
		},
		JSCache: &JSCache{
			entries: make(map[logger.Path]*jsCacheEntry),
		},
	}
}

type SourceIndexCache struct {
	mutex           sync.Mutex
	entries         map[logger.Path]uint32
	nextSourceIndex uint32
}

func (c *SourceIndexCache) LenHint() uint32 {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Add some extra room at the end for a new file or two without reallocating
	const someExtraRoom = 16
	return c.nextSourceIndex + someExtraRoom
}

func (c *SourceIndexCache) Get(path logger.Path) uint32 {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if sourceIndex, ok := c.entries[path]; ok {
		return sourceIndex
	}
	sourceIndex := c.nextSourceIndex
	c.nextSourceIndex++
	c.entries[path] = sourceIndex
	return sourceIndex
}