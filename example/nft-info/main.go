package main

import (
	"context"
	"fmt"

	"github.com/chaindead/tonutils-go/address"
	"github.com/chaindead/tonutils-go/liteclient"
	"github.com/chaindead/tonutils-go/ton"
	"github.com/chaindead/tonutils-go/ton/nft"
)

func main() {
	client := liteclient.NewConnectionPool()

	// connect to mainnet lite server
	err := client.AddConnection(context.Background(), "135.181.140.212:13206", "K0t3+IWLOXHYMvMcrGZDPs+pn58a17LFbnXoQkKc2xw=")
	if err != nil {
		panic(err)
	}

	// initialize ton api lite connection wrapper
	api := ton.NewAPIClient(client)

	nftAddr := address.MustParseAddr("EQC6KV4zs8TJtSZapOrRFmqSkxzpq-oSCoxekQRKElf4nC1I")
	item := nft.NewItemClient(api, nftAddr)

	nftData, err := item.GetNFTData(context.Background())
	if err != nil {
		panic(err)
	}

	var nftContent nft.ContentAny
	if nftData.CollectionAddress.Type() != address.NoneAddress {
		// get info about our nft's collection
		collection := nft.NewCollectionClient(api, nftData.CollectionAddress)
		collectionData, err := collection.GetCollectionData(context.Background())
		if err != nil {
			panic(err)
		}

		fmt.Println("Collection addr      :", nftData.CollectionAddress.String())
		switch content := collectionData.Content.(type) {
		case *nft.ContentOffchain:
			fmt.Println("    content offchain :", content.URI)
		case *nft.ContentOnchain:
			fmt.Println("    content onchain  :", content.GetAttribute("name"))
		}
		fmt.Println("    owner            :", collectionData.OwnerAddress.String())
		fmt.Println("    minted items num :", collectionData.NextItemIndex)
		fmt.Println()

		// get full nft's content url using collection method that will merge base url with nft's data
		nftContent, err = collection.GetNFTContent(context.Background(), nftData.Index, nftData.Content)
		if err != nil {
			panic(err)
		}
	}
	fmt.Println("NFT addr         :", nftAddr.String())
	fmt.Println("    initialized  :", nftData.Initialized)
	fmt.Println("    owner        :", nftData.OwnerAddress.String())
	fmt.Println("    index        :", nftData.Index)

	if nftData.Initialized {
		switch content := nftData.Content.(type) {
		case *nft.ContentOffchain:
			fmt.Println("    content :", content.URI)
			if nftContent != nil {
				fmt.Println("    full content :", nftContent.(*nft.ContentOffchain).URI)
			}
		case *nft.ContentOnchain:
			fmt.Println("    content name :", content.GetAttribute("name"))
		}
	} else {
		fmt.Println("    empty content")
	}
}
