package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type SyncDutiesResponse struct {
	Validators []string `json:"validators"`
}

// GetLatestSlot fetches the latest slot number from the beacon chain
func GetLatestSlot(nodeURL string) (int, error) {
	resp, err := http.Get(fmt.Sprintf("%s/eth/v1/beacon/headers", nodeURL))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var response struct {
		Data []struct {
			Header struct {
				Message struct {
					Slot string `json:"slot"`
				} `json:"message"`
			} `json:"header"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return 0, err
	}

	return strconv.Atoi(response.Data[0].Header.Message.Slot)
}

// GetSyncDuties retrieves a list of validators with sync committee duties for a given slot
func GetSyncDuties(c *gin.Context) {
	nodeURL := "https://radial-misty-butterfly.quiknode.pro/d71f751e03f2b6466202f2561941b6c1c0defd13"
	requestedSlot, err := strconv.Atoi(c.Param("slot"))
	if err != nil || requestedSlot < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid slot number"})
		return
	}

	latestSlot, err := GetLatestSlot(nodeURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch latest slot"})
		return
	}

	if requestedSlot > latestSlot {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Requested slot is too far in the future to have duties available"})
		return
	}

	resp, err := http.Get(fmt.Sprintf("%s/eth/v1/beacon/states/%d/sync_committees", nodeURL, requestedSlot))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch data"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		statusMap := map[int]string{
			http.StatusNotFound:            "Slot not found or no duties available",
			http.StatusInternalServerError: "Unexpected server error",
		}
		c.JSON(resp.StatusCode, gin.H{"error": statusMap[resp.StatusCode]})
		return
	}

	var duties struct {
		Data SyncDutiesResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&duties); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"slot":       requestedSlot,
		"validators": duties.Data.Validators,
	})
}
