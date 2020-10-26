package models

import "gopkg.in/mgo.v2"

var (
	globalS *mgo.Session
	dbname  string
)


type MgoConfig struct {
	Host           string `json:"host"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	DBName         string `json:"db_name"`
}

func Init(c MgoConfig) error {
	dialInfo := &mgo.DialInfo{
		Addrs:    []string{c.Host},
		Username: c.Username,
		Password: c.Password,
	}
	s, err := mgo.DialWithInfo(dialInfo)
	if err != nil {
		return err
	}
	err = s.Ping()
	if err != nil {
		return err
	}
	globalS = s
	dbname = c.DBName
	return nil
}

func connect(collection string) (*mgo.Session, *mgo.Collection) {
	s := globalS.Copy()
	c := s.DB(dbname).C(collection)
	return s, c
}

func Insert(collection string, docs ...interface{}) error {
	ms, c := connect(collection)
	defer ms.Close()
	return c.Insert(docs...)
}

func FindOne(collection string, query, selector, result interface{}) error {
	ms, c := connect(collection)
	defer ms.Close()
	return c.Find(query).Select(selector).One(result)
}

func FindAllSort(collection string, query interface{},result interface{}) error {
	ms, c := connect(collection)
	defer ms.Close()
	return c.Find(query).All(result)
}

func FindLastTipset(collection string, query, result interface{}) error {
	ms, c := connect(collection)
	defer ms.Close()
	return c.Find(query).Sort("-height").Limit(1).One(result)
}

func Upsert(collection string, selector, data interface{}) error {
	ms, c := connect(collection)
	defer ms.Close()
	_, err := c.Upsert(selector, data)
	return err
}