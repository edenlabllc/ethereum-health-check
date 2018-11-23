package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	proto "github.com/edenlabllc/eth_node_health_check/proto"
	"github.com/micro/cli"
	"github.com/micro/go-config"
	"github.com/micro/go-config/source/env"
	micro "github.com/micro/go-micro"
	"github.com/micro/go-micro/server"
)

var conf = config.NewConfig()

func ethScanURL(prefix, key string) string {
	return fmt.Sprintf("https://%s.etherscan.io/api?module=proxy&action=eth_blockNumber&apikey=%s", prefix, key)
}

func (h *ETHHealth) getLocalBlockNumber() (int64, error) {
	body := strings.NewReader(`{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":83}`)
	req, err := http.NewRequest("POST", h.node, body)
	if err != nil {
		// handle err
		fmt.Println("Cannot create POST request", err)
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	c := &http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		fmt.Println("http.Do() error", err)
		return 0, err
	}
	defer resp.Body.Close()

	if (resp.StatusCode >= 300) || (resp.StatusCode < 100) {
		return 0, fmt.Errorf("Server unavaible. Server status: %d", resp.StatusCode)
	}

	blockNumber, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Cannot parse response data", err)
		return 0, err
	}
	if string(blockNumber) == "" {
		fmt.Println("Empty local node response")
		return 0, errors.New("empty local node response")
	}

	var nodeResponse responseStruct
	err = json.Unmarshal(blockNumber, &nodeResponse)
	if err != nil {
		fmt.Println(err)
		return 0, err
	}

	res, err := strconv.ParseInt(nodeResponse.Result[2:], 16, 64)
	if err != nil {
		fmt.Println(err)
		return 0, err
	}

	return res, nil
	//return res
}

type responseStruct struct {
	ID      int    `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Result  string `json:"result"`
}

func (h *ETHHealth) getNetworkBlockNumber() (int64, error) {
	req, err := http.NewRequest("GET", h.etherScanURL, nil)
	if err != nil {
		fmt.Println("Cannot create POST request", err)
	}

	c := &http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		fmt.Println("http.Do() error", err)
		return 0, err
	}
	defer resp.Body.Close()

	if (resp.StatusCode >= 300) || (resp.StatusCode < 100) {
		return 0, fmt.Errorf("Server unavaible. Server status: %d", resp.StatusCode)
	}

	blockNumber, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Cannot parse response data", err)
		return 0, err
	}

	var etherscanResponse responseStruct
	err = json.Unmarshal(blockNumber, &etherscanResponse)
	if err != nil {
		fmt.Println(err)
		return 0, err
	}

	res, err := strconv.ParseInt(etherscanResponse.Result[2:], 16, 64)
	if err != nil {
		fmt.Println(err)
		return 0, err
	}

	return res, nil
}

// ETHHealth gRPC service struct
type ETHHealth struct {
	maxBlockDifference float64
	node               string
	etherScanURL       string
}

// Check Node health RPC handler
func (h *ETHHealth) Check(ctx context.Context, req *proto.Request, rsp *proto.Response) error {
	networkBlockNumber, err := h.getNetworkBlockNumber()
	if err != nil {
		fmt.Println(err)
		return err
	}

	localBlockNumber, err := h.getLocalBlockNumber()
	if err != nil {
		fmt.Println(err)
		return err
	}

	diff := math.Abs(float64(localBlockNumber - networkBlockNumber))
	if diff < h.maxBlockDifference {
		rsp.Health = true
		rsp.Diff = int32(diff)
		return nil
	}

	rsp.Health = false
	rsp.Diff = int32(diff)
	return nil
}

// Setup and the client
func runClient(service micro.Service) {
	// Create new greeter client
	health := proto.NewEthealthService("eth_health", service.Client())

	// Call the greeter
	rsp, err := health.Check(context.TODO(), &proto.Request{})
	if err != nil {
		fmt.Println(err)
		return
	}

	r, _ := json.Marshal(rsp)
	// Print response
	fmt.Println(string(r))
}

func logWrapper(fn server.HandlerFunc) server.HandlerFunc {
	start := time.Now()
	return func(ctx context.Context, req server.Request, rsp interface{}) error {
		err := fn(ctx, req, rsp)
		elapsed := time.Since(start)
		fmt.Printf("[%s] server request: %s\n", elapsed.String(), req.Method())
		return err
	}
}

// Required env
// MAXBLOCKDIFFERENCE = int amount diff between node and value from etherscan that is considorated health
// NODE_ADDR = str health check target url
// ETHERSCAN_PREFIX = str env of node and etherscan to look at
// ETHERSCAN_API_KEY = str API key
func main() {
	src := env.NewSource()
	conf.Load(src)

	service := micro.NewService(
		micro.Name("go.micro.api.ethealth"),
		micro.Version("latest"),
		micro.WrapHandler(logWrapper),

		// Setup some flags. Specify --run_client to run the client

		// Add runtime flags
		// We could do this below too
		micro.Flags(cli.BoolFlag{
			Name:  "run_client",
			Usage: "Launch the client",
		}),
	)

	// Init will parse the command line flags. Any flags set will
	// override the above settings. Options defined here will
	// override anything set on the command line.
	service.Init(
		// Add runtime action
		// We could actually do this above
		micro.Action(func(c *cli.Context) {
			if c.Bool("run_client") {
				runClient(service)
				os.Exit(0)
			}
		}),
	)
	// Setup the server
	handler := &ETHHealth{
		conf.Get("maxblockdifference").Float64(0),
		conf.Get("node", "addr").String("Add NODE_ADDR env"),
		ethScanURL(conf.Get("etherscan", "prefix").String("rinkeby"), conf.Get("etherscan", "api", "key").String("ADD API_KEY ENV"))}
	// Register handler
	if err := proto.RegisterEthealthHandler(
		service.Server(),
		handler); err != nil {
		fmt.Println(err)
	}
	// Run the server
	if err := service.Run(); err != nil {
		fmt.Println(err)
	}
}
