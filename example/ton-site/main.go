package main

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"github.com/chaindead/tonutils-go/adnl"
	"github.com/chaindead/tonutils-go/adnl/dht"
	rldphttp "github.com/chaindead/tonutils-go/adnl/rldp/http"
	"io"
	"log"
	"net"
	"net/http"
	"os"
)

func handler(writer http.ResponseWriter, request *http.Request) {
	_, _ = writer.Write([]byte("Hello, " + request.URL.Query().Get("name") +
		"\nThis TON site works natively using tonutils-go!"))
}

func main() {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}

	gateway := adnl.NewGateway(priv)
	err = gateway.StartClient()
	if err != nil {
		panic(err)
	}

	dhtClient, err := dht.NewClientFromConfigUrl(context.Background(), gateway, "https://ton-blockchain.github.io/testnet-global.config.json")
	if err != nil {
		panic(err)
	}

	mx := http.NewServeMux()
	mx.HandleFunc("/hello", handler)

	s := rldphttp.NewServer(loadKey(), dhtClient, mx)

	addr, err := rldphttp.SerializeADNLAddress(s.Address())
	if err != nil {
		panic(err)
	}

	log.Println("Listening on", addr+".adnl")
	s.SetExternalIP(net.ParseIP(getPublicIP()))
	if err = s.ListenAndServe(":9056"); err != nil {
		panic(err)
	}
}

func getPublicIP() string {
	req, err := http.Get("http://ip-api.com/json/")
	if err != nil {
		return err.Error()
	}
	defer req.Body.Close()

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return err.Error()
	}

	var ip struct {
		Query string
	}
	_ = json.Unmarshal(body, &ip)

	return ip.Query
}

func loadKey() ed25519.PrivateKey {
	file := "./key.txt"
	data, err := os.ReadFile(file)
	if err != nil {
		_, srvKey, err := ed25519.GenerateKey(nil)
		if err != nil {
			panic(err)
		}

		err = os.WriteFile(file, []byte(hex.EncodeToString(srvKey.Seed())), 555)
		if err != nil {
			panic(err)
		}

		return srvKey
	}

	dec, err := hex.DecodeString(string(data))
	return ed25519.NewKeyFromSeed(dec)
}
