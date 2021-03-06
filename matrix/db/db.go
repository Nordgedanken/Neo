package db

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/dgraph-io/badger"
	"github.com/shibukawa/configdir"
)

var UserDB *badger.DB
var CacheDB *badger.DB
var onceCache sync.Once
var onceUser sync.Once

// OpenCacheDB opens or generates the Database file for settings and Cache
func OpenCacheDB() (db *badger.DB, err error) {
	onceCache.Do(func() {
		// Open the data.db file. It will be created if it doesn't exist.
		configDirs := configdir.New("Nordgedanken", "Morpheus")
		if _, StatErr := os.Stat(filepath.ToSlash(configDirs.QueryFolders(configdir.Global)[0].Path) + "/data/"); os.IsNotExist(StatErr) {
			MkdirErr := os.MkdirAll(filepath.ToSlash(configDirs.QueryFolders(configdir.Global)[0].Path)+"/data/", 0666)
			if MkdirErr != nil {
				err = MkdirErr
				return
			}
		}
		if _, StatErr := os.Stat(filepath.ToSlash(configDirs.QueryFolders(configdir.Global)[0].Path) + "/data/cache/"); os.IsNotExist(StatErr) {
			MkdirErr := os.MkdirAll(filepath.ToSlash(configDirs.QueryFolders(configdir.Global)[0].Path)+"/data/cache/", 0666)
			if MkdirErr != nil {
				err = MkdirErr
				return
			}
		}
		opts := badger.DefaultOptions
		opts.SyncWrites = false
		opts.Dir = filepath.ToSlash(configDirs.QueryFolders(configdir.Global)[0].Path) + "/data/cache"
		opts.ValueDir = filepath.ToSlash(configDirs.QueryFolders(configdir.Global)[0].Path) + "/data/cache"

		expDB, DBErr := badger.Open(opts)
		if DBErr != nil {
			err = DBErr
			return
		}
		CacheDB = expDB
	})

	if CacheDB == nil {
		err = errors.New("missing CacheDB")
		return
	}

	db = CacheDB
	return
}

// OpenUserDB opens or generates the Database file for settings and Cache
func OpenUserDB() (db *badger.DB, err error) {
	onceUser.Do(func() {
		// Open the data.db file. It will be created if it doesn't exist.
		configDirs := configdir.New("Nordgedanken", "Morpheus")
		if _, StatErr := os.Stat(filepath.ToSlash(configDirs.QueryFolders(configdir.Global)[0].Path) + "/data/"); os.IsNotExist(StatErr) {
			MkdirErr := os.MkdirAll(filepath.ToSlash(configDirs.QueryFolders(configdir.Global)[0].Path)+"/data/", 0666)
			if MkdirErr != nil {
				err = MkdirErr
				return
			}
		}
		if _, StatErr := os.Stat(filepath.ToSlash(configDirs.QueryFolders(configdir.Global)[0].Path) + "/data/user/"); os.IsNotExist(StatErr) {
			MkdirErr := os.MkdirAll(filepath.ToSlash(configDirs.QueryFolders(configdir.Global)[0].Path)+"/data/user/", 0666)
			if MkdirErr != nil {
				err = MkdirErr
				return
			}
		}
		opts := badger.DefaultOptions
		opts.SyncWrites = false
		opts.Dir = filepath.ToSlash(configDirs.QueryFolders(configdir.Global)[0].Path) + "/data/user"
		opts.ValueDir = filepath.ToSlash(configDirs.QueryFolders(configdir.Global)[0].Path) + "/data/user"

		expDB, DBErr := badger.Open(opts)
		if DBErr != nil {
			err = DBErr
			return
		}

		UserDB = expDB
	})

	if UserDB == nil {
		err = errors.New("missing UserDB")
		return
	}

	db = UserDB
	return
}

// CacheMessageEvents writes message infos into the cache into the defined room
func CacheMessageEvents(id, sender, roomID, message string, timestamp int64) (err error) {
	db, DBOpenErr := OpenCacheDB()
	if DBOpenErr != nil {
		err = DBOpenErr
		return
	}

	// Update cache
	DBerr := db.Update(func(txn *badger.Txn) error {
		DBSetIDErr := txn.Set([]byte("room|"+roomID+"|messages|"+id+"|id"), []byte(id))
		if DBSetIDErr != nil {
			return DBSetIDErr
		}

		DBSetSenderErr := txn.Set([]byte("room|"+roomID+"|messages|"+id+"|sender"), []byte(sender))
		if DBSetSenderErr != nil {
			return DBSetSenderErr
		}

		DBSetMessageErr := txn.Set([]byte("room|"+roomID+"|messages|"+id+"|messageString"), []byte(message))
		if DBSetMessageErr != nil {
			return DBSetMessageErr
		}

		timestampString := strconv.FormatInt(timestamp, 10)
		DBSeTimestampErr := txn.Set([]byte("room|"+roomID+"|messages|"+id+"|timestamp"), []byte(timestampString))
		return DBSeTimestampErr
	})

	if DBerr != nil {
		err = DBerr
		return
	}

	return
}
