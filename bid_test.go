package bid

import(
	"fmt"
	"time"

	"github.com/garyburd/redigo/redis"
)

func ExampleSetConfig(){
	var err error

	pool := &redis.Pool{
		MaxIdle: 100,
		MaxActive: 1000,
		IdleTimeout: 1000,
		Wait: true,
	}
		conn, err := redis.Dial("tcp", ":6379")
		if err != nil {
			return
		}
	defer pool.Close()

	t := time.Now()
	err = setConfig(t, pool)
	if err != nil{
		return
	}

	a, err := conn.Do("GET", "licenses")
	if err != nil{
                return
        }
	b, err := conn.Do("GET", "startPrice")
	if err != nil{
                return
        }
	c, err := conn.Do("HGETALL", "time")
	if err != nil{
                return
        }

	fmt.Printf("a:%v\n", a)
	fmt.Printf("b:%v\n", b)
	fmt.Printf("c:%v\n", c)
}
