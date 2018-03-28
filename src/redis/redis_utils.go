package redis

import (
  "fmt"
  "github.com/gomodule/redigo/redis"
)

func Ping() error {

  conn := Pool.Get()
  defer conn.Close()

  _, err := redis.String(conn.Do("PING"))
  if err != nil {
    return fmt.Errorf("cannot 'PING' db: %v", err)
  }
  return nil
}

func Get(key string) (interface{}, error) {

  conn := Pool.Get()
  defer conn.Close()

  data, err := conn.Do("GET", key)
  if err != nil {
    return data, fmt.Errorf("error getting key %s: %v", key, err)
  }
  return data, err
}

func GetField(key string, field string) (interface{}, error) {

  conn := Pool.Get()
  defer conn.Close()

  data, err := conn.Do("HGET", key, field)
  if err != nil {
    return data, fmt.Errorf("error getting key %s: %v", key, err)
  }
  return data, err
}

func Set(key string, value interface{}) error {

  /*
  []byte                  Sent as is
string                  Sent as is
int, int64              strconv.FormatInt(v)
float64                 strconv.FormatFloat(v, 'g', -1, 64)
bool
*/
  conn := Pool.Get()
  defer conn.Close()

  _, err := conn.Do("SET", key, value)

  if err != nil {
    return fmt.Errorf("error setting key %s: %v", key, err)
  }
  return err
}

// Sets field in the hash stored at key to value.
// If key does not exist, a new key holding a hash is created.
// If field already exists in the hash, it is overwritten.
func SetField(key string, field string, value interface{}) error {
  /*
  []byte                  Sent as is
string                  Sent as is
int, int64              strconv.FormatInt(v)
float64                 strconv.FormatFloat(v, 'g', -1, 64)
bool
*/

  conn := Pool.Get()
  defer conn.Close()

  // Use HMSET for storing multiple fields... HSET for one
  /*
  var err error
  switch value.(type) {
  case []byte:
      _, err = conn.Do("HSET", key, field,  value.([]byte))
  case string:
      _, err = conn.Do("HSET", key, field,  value.(string))
  case int:
      _, err = conn.Do("HSET", key, field,  value.(int))
  case float64:
      _, err = conn.Do("HSET", key, field,  value.(float64))
  case bool:
      _, err = conn.Do("HSET", key, field,  value.(bool))
  default:
    _, err = conn.Do("HSET", key, field,  value)
  } */

  _, err := conn.Do("HSET", key, field,  value)

  if err != nil {
    return fmt.Errorf("error setting key %s: %v", key, err)
  }
  return err
}

func HExists(key string, field string) (bool, error) {

  conn := Pool.Get()
  defer conn.Close()

  ok, err := redis.Bool(conn.Do("HEXISTS", key, field))
  if err != nil {
    return ok, fmt.Errorf("error checking if key %s exists: %v", key, err)
  }
  return ok, err
}

func Exists(key string) (bool, error) {

  conn := Pool.Get()
  defer conn.Close()

  ok, err := redis.Bool(conn.Do("EXISTS", key))
  if err != nil {
    return ok, fmt.Errorf("error checking if key %s exists: %v", key, err)
  }
  return ok, err
}

func Delete(key string) error {

  conn := Pool.Get()
  defer conn.Close()

  _, err := conn.Do("DEL", key)
  return err
}

func GetKeys(pattern string) ([]string, error) {

  conn := Pool.Get()
  defer conn.Close()

  iter := 0
  keys := []string{}
  for {
    arr, err := redis.Values(conn.Do("SCAN", iter, "MATCH", pattern))
    if err != nil {
      return keys, fmt.Errorf("error retrieving '%s' keys", pattern)
    }

    iter, _ = redis.Int(arr[0], nil)
    k, _ := redis.Strings(arr[1], nil)
    keys = append(keys, k...)

    if iter == 0 {
      break
    }
  }

  return keys, nil
}

func Incr(counterKey string) (int, error) {

  conn := Pool.Get()
  defer conn.Close()

  return redis.Int(conn.Do("INCR", counterKey))
}

//Increment the specified field of a hash stored at key,
// and representing a floating point number, by the specified increment
func HIncrByFloat(counterKey string, field string, by float64) (interface{}, error) {

  conn := Pool.Get()
  defer conn.Close()

  return conn.Do("HINCRBYFLOAT", counterKey, field, by)
}
