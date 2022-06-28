package main

import (
	"applemint-go/crawl"
	"applemint-go/crud"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
)

func handleImgurAnalyzeRequest(w http.ResponseWriter, r *http.Request) {
	log.Println("handleImgurAnalyzeRequest:", r.URL.Path)
	imgurLink := r.URL.Query().Get("link")
	if imgurLink == "" {
		log.Println("handleImgurAnalyzeRequest: missing link")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	images, err := crawl.HandleImgurLink(imgurLink)
	if err != nil {
		log.Println("handleImgurAnalyzeRequest:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Println("handleImgurAnalyzeRequest:", images)
	w.WriteHeader(http.StatusOK)
	json, err := json.Marshal(images)
	if err != nil {
		log.Println("handleImgurAnalyzeRequest:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(json)
}

func handleCollectionInfoRequest(w http.ResponseWriter, r *http.Request) {
	collection := mux.Vars(r)["collection"]
	totalCount, GroupInfos, err := crud.GetCollectionInfo(collection)
	if err != nil {
		fmt.Fprintf(w, "Error getting collection info: %s", err)
		return
	}
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"totalCount": totalCount,
		"groupInfos": GroupInfos,
	})
}

func handleClearCollectionRequest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	target := mux.Vars(r)["target"]
	delCnt := crud.ClearCollection(target)
	if delCnt > 0 {
		fmt.Fprintf(w, "Deleted %d items from collection %s", delCnt, target)
	} else {
		fmt.Fprintf(w, "Collection %s is empty", target)
	}
}

func handleCrawlRequest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	switch expression := mux.Vars(r)["target"]; expression {
	case "bp":
		json.NewEncoder(w).Encode(crawl.CrawlBP())
	case "isg":
		json.NewEncoder(w).Encode(crawl.CrawlISG())
	}
}

func handleMoveItemRequest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	targetId := mux.Vars(r)["id"]
	target_coll := r.URL.Query().Get("target")
	origin_coll := r.URL.Query().Get("origin")
	if target_coll == "" || origin_coll == "" {
		fmt.Fprintf(w, "Missing parameters")
		return
	}
	err := crud.MoveItem(targetId, origin_coll, target_coll)
	if err != nil {
		fmt.Fprintf(w, "Error moving item: %s", err)
		return
	}
	fmt.Fprintf(w, "Item moved from %s to %s", origin_coll, target_coll)
}

func handleKeepItemRequest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	targetId := mux.Vars(r)["id"]
	item := crud.Item{}
	err := json.NewDecoder(r.Body).Decode(&item)
	if err != nil {
		fmt.Fprintf(w, "Error decoding item: %s", err)
		return
	}
	updateCnt := crud.UpdateItem(targetId, "new", item)
	if updateCnt > 0 {
		fmt.Fprintf(w, "Updated %d items\n", updateCnt)
	} else {
		fmt.Fprintf(w, "No items updated")
		return
	}
	err = crud.MoveItem(targetId, "new", "keep")
	if err != nil {
		fmt.Fprintf(w, "Error moving item: %s", err)
		return
	}

	fmt.Fprintf(w, "Updated Item moved from new to keep")
}

func handleItemsRequest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	target := mux.Vars(r)["collection"]
	cursor, err := strconv.Atoi(r.URL.Query().Get("cursor"))
	if err != nil {
		cursor = 0
	}

	domain := r.URL.Query().Get("domain")
	path := r.URL.Query().Get("path")

	items, err := crud.GetItems(target, int64(cursor), domain, path)
	if err != nil {
		fmt.Fprintf(w, `{"Error getting items": "%s"}`, err)
		return
	}

	json.NewEncoder(w).Encode(items)
}

func handleItemRequest(w http.ResponseWriter, r *http.Request) {

	w.WriteHeader(http.StatusOK)
	targetId := mux.Vars(r)["id"]
	targetCollection := mux.Vars(r)["collection"]
	switch r.Method {
	case "GET":
		item, err := crud.GetItem(targetId, targetCollection)
		if err != nil {
			fmt.Fprintf(w, "Error getting item: %s", err)
			return
		}
		json.NewEncoder(w).Encode(item)

	case "POST":
		item := crud.Item{}
		err := json.NewDecoder(r.Body).Decode(&item)
		if err != nil {
			fmt.Fprintf(w, "Error decoding item: %s", err)
			return
		}
		updateCnt := crud.UpdateItem(targetId, targetCollection, item)
		if updateCnt > 0 {
			fmt.Fprintf(w, "Updated %d items from collection %s", updateCnt, targetCollection)
		} else {
			fmt.Fprintf(w, "Collection %s is empty", targetCollection)
		}
	case "DELETE":
		delCnt := crud.DeleteItem(targetId, targetCollection)
		fmt.Println("Deleted", delCnt, "items")
		if delCnt > 0 {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"deleted": delCnt,
			})
			// fmt.Fprintf(w, "{\"msg\": \"item deleted from %s -> %s\"}", targetCollection, targetId)
		} else {
			fmt.Fprintf(w, "{\"error\": \"cannot find item from %s -> %s\"}", targetCollection, targetId)
		}
	}
}

func handleDropboxRequest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	path := r.URL.Query().Get("path")
	url := r.URL.Query().Get("url")
	if path == "" || url == "" {
		fmt.Fprintf(w, "Missing parameters")
		return
	}
	err := crud.SendToDropbox(path, url)
	if err != nil {
		fmt.Fprintf(w, "Error sending to dropbox: %s", err)
		return
	}
}

func handleRaindropCollectionRequest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	collections, err := crud.GetCollectionFromRaindrop()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "%s", string(collections))
}

func handleRaindropRequest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	collectionId := mux.Vars(r)["collectionId"]
	item := crud.Item{}
	err := json.NewDecoder(r.Body).Decode(&item)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if collectionId == "" {
		fmt.Fprintf(w, "Missing parameters")
		return
	}

	raindropResp, err := crud.SendToRaindrop(item, collectionId)
	if err != nil {
		fmt.Fprintf(w, "Error sending to raindrop: %s", err)
		return
	}
	fmt.Fprintf(w, "%s", string(raindropResp))
}

func handleBookmarkRequest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	switch r.Method {
	case "GET":
		log.Println("handle bookmark list get")
		bookmarks, err := crud.GetBookmarkList()
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(bookmarks)

	case "POST":
		log.Print("handleSendToBookmark")
		source := r.URL.Query().Get("from")
		path := r.URL.Query().Get("path")
		log.Printf("source: %s, path: %s", source, path)

		if source == "" || path == "" {
			log.Print("missing source or path")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		item := crud.Item{}
		err := json.NewDecoder(r.Body).Decode(&item)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		result, err := crud.SendToBookmark(item, source, path)
		log.Println(result)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	name := os.Getenv(("NAME"))
	if name == "" {
		name = "World"
	}
	fmt.Fprintf(w, "Hello %s!", name)
}
