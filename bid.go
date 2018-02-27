package bid

import(
       "fmt"
       "time"

       "github.com/garyburd/redigo/redis"
      )
//在server启动时写入拍卖初始数据
func setConfig(t time.Time, pool redis.Pool)(err error) {

	conn := pool.Get()
	defer conn.Close()

	start := t.Add(time.Second * 1)
	second := start.Add(time.Second * 1)
	end := second.Add(time.Second * 2)
	start1 := start.Unix()
	second1 := second.Unix()
	end1 := end.Unix()
	conn.Do("MULTI")
	conn.Do("SET", "licenses", "10000")
	conn.Do("SET", "startPrice", "80000")
	conn.Do("HSET", "time", "startTime", start1, "secondTime", second1, "endTime", end1)
        _, err = conn.Do("EXEC")
	if err != nil{
		fmt.Errorf("HSET Error: %v", err)
		return err
	}
	return nil
}

//返回目前所在的的阶段
func getPhase(t time.Time, pool redis.Pool)(i int, err error) {

	type phase struct{
		startTime int64 `redis:"startTime"`
		secondTime int64 `redis:"secondTime"`
		endTime int64 `redis:"endTime"`
	}

	conn := pool.Get()
        defer conn.Close()

	//将redis中的数据传入结构体
	values, err := redis.Values(conn.Do("HGETALL", "time"))
	if err != nil {
		fmt.Errorf("Do Error: %v", err)
		return i, err
	}

	p := phase{}
	if err = redis.ScanStruct(values, &p); err != nil {
		fmt.Errorf("ScanStruct Error: %v", err)
		return i, err
	}
	//判断目前时间处于第几阶段
	str := time.Unix(p.startTime, 0)
	if t.Before(str) == true {
		i = 0
		return i, nil
	}

	str = time.Unix(p.secondTime, 0)
	if t.Before(str) == true {
		i = 1
		return i, nil
	}

	str = time.Unix(p.endTime, 0)
	if t.Before(str) == true {
		i = 2
		return i, nil
	}

	if t.After(str) == true {
		i = 3
		return i, nil
	}

	fmt.Errorf("Unknow time", err)
	return i, err
}

//取得参加拍卖的用户数量
func buyerNbr(pool redis.Pool)(buyerNbr int64, err error){

	conn := pool.Get()
        defer conn.Close()

	buyerNbr, err = redis.Int64(conn.Do("GET", "buyerNbr"))
        if err != nil {
                return buyerNbr, err
        }

	return buyerNbr, nil
}

//返回各阶段的时间
func getTimes(pool redis.Pool)(startTime time.Time, secondTime time.Time, endTime time.Time, err error){

	conn := pool.Get()
        defer conn.Close()

	t1, err := redis.Int64(conn.Do("HGET", "time", "startTime"))
	        if err != nil{
                fmt.Errorf("HGET Error: %v", err)
                return startTime, secondTime, endTime, err
        }
	t2, err := redis.Int64(conn.Do("HGET", "time", "secondTime"))
	        if err != nil{
                fmt.Errorf("HGET Error: %v", err)
                return startTime, secondTime, endTime, err
        }
	t3, err := redis.Int64(conn.Do("HGET", "time", "endTime"))
	        if err != nil{
                fmt.Errorf("HGET Error: %v", err)
                return startTime, secondTime, endTime, err
        }

	startTime = time.Unix(t1, 0)
	secondTime = time.Unix(t2, 0)
	endTime = time.Unix(t3, 0)
	return startTime, secondTime, endTime, nil
}

//写入出价
func bid(bid, int64, nbrPlate int64, i int, rsvPrice int64, t time.Time, pool redis.Pool)(err error) {

        conn := pool.Get()
        defer conn.Close()

	//判断是否还有竞拍机会
	chance, err := redis.Int(conn.Do("HLEN", nbrPlate))
	if err != nil {
		fmt.Errorf("Redis Error: %v", err)
	return err
	}
	switch i {
	    case 0:
		fmt.Errorf("拍卖还未开始", err)
		return err

	    case 1:
		if chance == 0 {
		//判断是否价格为最低成交价加减三百的区间内
			if (bid - rsvPrice)>300 || (rsvPrice - bid)<(-300){
				fmt.Errorf("Invalid Price: %v", err)
				return err
			}
			str := t.Format("15:04:05")
			tnano := t.UnixNano()
			key1 := fmt.Sprintf("result:%v", bid)
			key2 := fmt.Sprintf("bid:%v", bid)
			//通过pipeline进行写数据
			conn.Do("MULTI")
			conn.Do("ZADD", "price", bid, bid)
			conn.Do("HSET", nbrPlate, str, bid)
			conn.Do("ZADD", key1, tnano, nbrPlate)
			conn.Do("INCR", key2)
			conn.Do("INCR", "buyerNbr")
			_, err := conn.Do("EXEC")
			if err != nil{
				fmt.Errorf("Pipeline Failed: %v", err)
				return err
			}
			return nil
		}
		fmt.Errorf("您当前阶段出价次数已用完")
		return err

           case 2:
		if chance == 1 {
                //判断是否价格为最低成交价加减三百的区间内
                        if (bid - rsvPrice)>300 || (rsvPrice - bid)<(-300){
                                fmt.Errorf("Invalid Price: %v", err)
                                return err
                        }
                        str := t.Format("15:04:05")
                        tnano := t.UnixNano()
                        key1 := fmt.Sprintf("result:%v", bid)
                        key2 := fmt.Sprintf("bid:%v", bid)
                        //通过pipeline进行写数据
                        conn.Do("MULTI")
                        conn.Do("ZADD", "price", bid, bid)
                        conn.Do("HSET", nbrPlate, str, bid)
                        conn.Do("ZADD", key1, tnano, nbrPlate)
                        conn.Do("INCR", key2)
                        _, err = conn.Do("EXEC")
                        if err != nil{
                                fmt.Errorf("Pipeline Failed: %v", err)
                                return err
                        }
                        return nil
                }
                fmt.Errorf("您当前阶段出价次数已用完")
                return err

           case 3:
		fmt.Errorf("拍卖已经结束")
		return err
	}
	fmt.Errorf("未知时段")
	return err
}
/*
//提取拍卖者信息
func getBuyerInfo(nbrPlate int64, pool redis.Pool)(time1 time.Time, price1 int64, time2 time.Time, price2 int64){
	var(

	)

	type buyer struct{
		time1 time.Time
		bid1  int64
		time2 time.Time
		bid2  int64
	}

	conn := pool.Get()
	defer conn.Close()



}
*/

//计算通用最低成交价
func reservePrice(i int, pool redis.Pool)(rsvprice int64, err error){
	var(
	    prices []int64
	    sum int64
	    sub int64
	    rsvPrice int64
            buyerNbr int64
	)

	conn := pool.Get()
        defer conn.Close()

	//判断目前阶段，根据不同阶段执行操作
	if i == 0 {
		rsvPrice, err = redis.Int64(conn.Do("GET", "startingPrice"))
		if err != nil {
			fmt.Errorf("GET Error: %v", err)
			return rsvPrice, err
		}
	}

	//取价格，再去用户提交的价格里查人数
	prices, err = redis.Int64s(conn.Do("ZREVRANGE", "price", "0", "-1"))
        if err != nil {
		fmt.Errorf("Sort Error: %v", err)
		return rsvPrice, err
        }

	//提取同价位的人数
        for _, rsvPrice = range prices {
		sub, err = redis.Int64(conn.Do("GET", rsvPrice))
                if err != nil {
			fmt.Errorf("GET Error: %v", err)
			return rsvPrice, err
                }
                break
        }

	//取价格次高者开始计算最低成交价
	licenses, err := redis.Int64(conn.Do("GET", "licenses"))
	if err != nil {
		fmt.Errorf("GET Error: %v", err)
		return rsvPrice, err
        }

        for _, rsvPrice = range prices {
		buyerNbr, err = redis.Int64(conn.Do("GET", rsvPrice))
		if err != nil{
			fmt.Errorf("GET Error: %v", err)
			return rsvPrice, err
		}
                sum += buyerNbr
		if (sum - sub) >= licenses {
                        return rsvPrice, nil
                }
        }
	return rsvPrice, nil
}

//统计竞拍中标者，将ID写入redis数据库
func result(pool redis.Pool)(err error){

	var(
		maxPrice int64
		rsvPrice int64
		buyerNbr int64
		sum int64
		sub int64
	)

	conn := pool.Get()
	defer conn.Close()

        //取价格，再去用户提交的价格里查人数
	prices, err := redis.Int64s(conn.Do("ZREVRANGE", "price", "0", "-1"))
        if err != nil {
                fmt.Errorf("ZREVRANGE Error: %v", err)
                return err
        }

        //提取同价位的人数
	for _, maxPrice = range prices {
		sub, err = redis.Int64(conn.Do("GET", rsvPrice))
                if err != nil {
                        fmt.Errorf("GET Error: %v", err)
                        return err
                }
                break
        }

        //取价格次高者开始计算最低成交价
        licenses, err := redis.Int64(conn.Do("GET", "licenses"))
        if err != nil {
                fmt.Errorf("GET Error: %v", err)
                return err
        }

        for _, rsvPrice = range prices {
                buyerNbr, err = redis.Int64(conn.Do("GET", rsvPrice))
                if err != nil{
                        fmt.Errorf("GET Error: %v", err)
                        return err
                }
		if rsvPrice == maxPrice{
			continue
		}
		//从次高价开始每个最低成交价之上价位的id写入集合中
		idSli, err := redis.Strings(conn.Do("ZRANGE", "ss:%v", "0", "-1"))
		if err != nil{
			fmt.Errorf("ZRANGE Error: %v", err)
			return err
		}
		for i := 0; i < len(idSli); i++ {
			_, err = conn.Do("SADD", "idList", idSli[i])
			if err != nil{
				fmt.Errorf("SADD Error: %v", err)
				return err
			}
		}
		//在最低成交价中取中标者id
                sum += buyerNbr
                if (sum - sub) >= licenses {
			k := (sum - sub - licenses - 1)
			idSli, err = redis.Strings(conn.Do("ZRANGE", "ss:%v", rsvPrice, "0", k))
	                if err != nil{
				fmt.Errorf("ZRANGE Error: %v", err)
                        return err
			}

			for i := 0; i < len(idSli); i++ {
                                _, err = conn.Do("SADD", "idList", idSli[i])
                                if err != nil{
					fmt.Errorf("SADD Error: %v", err)
					return err
				}

			}
			return nil
                }

		for i := 0; i < len(idSli); i++ {
                                _, err = conn.Do("SADD", "idList", idSli[i])
                                if err != nil{
                                        fmt.Errorf("SADD Error: %v", err)
                                        return err
                                }
                }

        }
        return nil
}

//判断用户是否中标
func whetherSuccess(nbrPlate int64, pool redis.Pool)(status bool, err error){

	conn := pool.Get()
	defer conn.Close()

	status, err = redis.Bool(conn.Do("SISMEMBER", "idList", nbrPlate))
	if err != nil{
		fmt.Errorf("Bool Error: %v", err)
		return status, err
	}
	return status, nil
}
