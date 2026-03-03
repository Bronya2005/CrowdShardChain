package badger

import (
	"errors"

	"github.com/dgraph-io/badger/v4"
)

type Storage struct {
	db *badger.DB
}

func Open(path string) (*Storage, error) {
	opt := badger.DefaultOptions(path).WithLogger(nil)
	db, err := badger.Open(opt)
	if err != nil {
		return nil, errors.New("打开Badger失败")
	}
	return &Storage{db: db}, nil
}

func (s *Storage) Close() error {
	if s == nil || s.db == nil {
		return errors.New("Badger未初始化")
	}
	return s.db.Close()
}

func (s *Storage) Get(key []byte) ([]byte, error) {
	if len(key) == 0 {
		return nil, errors.New("Get失败：key不能为空")
	}
	var out []byte
	err := s.db.View(func(txn *badger.Txn) error {
		it, err := txn.Get(key)
		if err != nil {
			return err
		}
		val, err := it.ValueCopy(nil)
		if err != nil {
			return err
		}
		out = val
		return nil
	})
	if err != nil {
		return nil, errors.New("Get失败：读取数据库失败")
	}
	return out, nil
}

func (s *Storage) Set(key []byte, val []byte) error {
	if len(key) == 0 {
		return errors.New("Set失败：key不能为空")
	}
	if val == nil {
		val = []byte{}
	}
	err := s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, val)
	})
	if err != nil {
		return errors.New("Set失败：写入数据库失败")
	}
	return nil
}

func (s *Storage) Delete(key []byte) error {
	if len(key) == 0 {
		return errors.New("Delete失败：key不能为空")
	}
	err := s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
	if err != nil {
		return errors.New("Delete失败：删除失败")
	}
	return nil
}

func (s *Storage) ScanPrefix(prefix []byte, fn func(k, v []byte) error) error {
	if fn == nil {
		return errors.New("ScanPrefix失败：回调不能为空")
	}
	return s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := item.KeyCopy(nil)
			v, err := item.ValueCopy(nil)
			if err != nil {
				return errors.New("ScanPrefix失败：读取value失败")
			}
			if err := fn(k, v); err != nil {
				return err
			}
		}
		return nil
	})
}
