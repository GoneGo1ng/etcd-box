package manager

import (
	"context"
	"fmt"
	"go.etcd.io/etcd/clientv3"
	"testing"
)

func TestEtcd(t *testing.T) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints: []string{"http://172.16.4.6:2379"},
		Username:  "root",
		Password:  "hzsun@310012",
	})
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	/*start := time.Now()
	_, err = client.Get(context.Background(), "/", clientv3.WithPrefix(), clientv3.WithKeysOnly())
	fmt.Println(err)
	fmt.Println(time.Since(start).String())*/
	// client.Put(context.Background(),"/a/b/c1","foo")
	resp, err := client.Get(context.Background(), "/foo",
		clientv3.WithPrefix(), clientv3.WithKeysOnly())
	fmt.Println(err)
	if resp != nil && resp.Kvs != nil {
		for _, v := range resp.Kvs {
			fmt.Println(string(v.Key), string(v.Value))
		}
	}
}
