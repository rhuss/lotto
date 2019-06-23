package main // import "github.com/rhuss/lotto"

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"

	"github.com/cloudevents/sdk-go"
)

var (
	port = 8080
	path = "/"

	sequence = 0
)

type LottoRecommendation struct {
	Sequence int   `json:"sequence"`
	Seed     int64 `json:"seed"`
	Tip      []int `json:"tip"`
}

func (l *LottoRecommendation) Draw(seed int64) {
	sequence++
	l.Sequence = sequence
	l.Seed = seed
	if seed != 0 {
		rand.Seed(seed)
	}
	l.Tip = make([]int, 6)
	for i := 0; i < 6; i++ {
		l.Tip[i] = rand.Intn(49) + 1
	}
	sort.Slice(l.Tip, func(i, j int) bool { return l.Tip[i] < l.Tip[j] })
}

func main() {
	mode := os.Getenv("MODE")
	if mode == "rest" {
		os.Exit(startRestService())
	}
	os.Exit(startEventService())
}

func startEventService() int {
	ctx := context.Background()

	t, err := cloudevents.NewHTTPTransport(
		cloudevents.WithPort(port),
		cloudevents.WithPath(path),
	)
	if err != nil {
		log.Fatalf("failed to create transport: %s", err.Error())
	}
	c, err := cloudevents.NewClient(t,
		cloudevents.WithUUIDs(),
		cloudevents.WithTimeNow(),
	)
	if err != nil {
		log.Fatalf("failed to create client: %s", err.Error())
	}

	log.Printf("Event service listening on :%d%s\n", port, path)

	if err := c.StartReceiver(ctx, addToEvent); err != nil {
		log.Fatalf("failed to start receiver: %s", err.Error())
	}

	<-ctx.Done()
	return 0
}

func addToEvent(ctx context.Context, event cloudevents.Event, resp *cloudevents.EventResponse) error {
	fmt.Printf("Got Event Context: %+v\n", event.Context)
	data := &LottoRecommendation{}
	data.Draw(eventToInt(event))

	fmt.Printf("Got Event Data: %+v\n", event.Data)
	fmt.Printf("Adding Lotto tip: %+v\n", data)
	fmt.Printf("Got Transport Context: %+v\n", cloudevents.HTTPTransportContextFrom(ctx))
	fmt.Printf("----------------------------\n")

	r := cloudevents.NewEvent()
	r.SetSource("/lotto")
	r.SetType("samples.http.mod3")
	r.SetID(event.Context.GetID())
	if err := r.SetData(data); err != nil {
		return err
	}
	resp.RespondWith(200, &r)

	return nil
}

func eventToInt(event cloudevents.Event) int64 {
	sum := md5.Sum([]byte(fmt.Sprintf("%+v", event)))
	var ret int64
	buf := bytes.NewBuffer(sum[:])
	binary.Read(buf, binary.BigEndian, &ret)
	return ret
}

func startRestService() int {
	http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		seedS := r.URL.Query().Get("seed")
		var seed int64
		var err error
		if seedS != "" {
			seed, err = strconv.ParseInt(seedS, 10, 64)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Invalid seed provided"))
				return
			}
		}
		lotto := &LottoRecommendation{}
		lotto.Draw(seed)
		err = json.NewEncoder(w).Encode(lotto)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("Error while encoding %+v", lotto)))
			return
		}
		w.Header().Set("Content-Type", "application/json")
	})

	log.Printf("Rest service listening on :%d%s\n", port, path)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		return 1
	}
	return 0
}
