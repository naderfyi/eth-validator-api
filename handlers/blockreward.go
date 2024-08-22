package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// BlockRewardResponse defines the structure of the response for the block reward endpoint
type BlockRewardResponse struct {
	Status string `json:"status"`
}

// BlockData represents the structure for block data returned by the beacon node
type BlockData struct {
	Data struct {
		Message struct {
			Body struct {
				ProposerIndex int      `json:"proposer_index"`
				Transactions  []string `json:"transactions"`
			} `json:"body"`
		} `json:"message"`
	} `json:"data"`
}

// FetchBlockData retrieves block data for a given slot
func FetchBlockData(slot int) (*BlockData, error) {
	nodeURL := "https://radial-misty-butterfly.quiknode.pro/d71f751e03f2b6466202f2561941b6c1c0defd13"
	resp, err := http.Get(fmt.Sprintf("%s/eth/v2/beacon/blocks/%d", nodeURL, slot))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch block data, status code: %d", resp.StatusCode)
	}

	var blockData BlockData
	if err := json.NewDecoder(resp.Body).Decode(&blockData); err != nil {
		return nil, err
	}

	return &blockData, nil
}

// IsMEVBlock determines if a block was produced using an MEV relay by checking known MEV relay addresses
func IsMEVBlock(transactions []string) bool {
	// List of known MEV relays for Ethereum Mainnet
	mevRelays := []string{
		"a15b52576bcbf1072f4a011c0f99f9fb6c66f3e1ff321f11f461d15e31b1cb359caa092c71bbded0bae5b5ea401aab7e", // Aestus
		"a7ab7a996c8584251c8f925da3170bdfd6ebc75d50f5ddc4050a6fdc77f2a3b5fce2cc750d0865e05d7228af97d69561", // Agnostic Gnosis
		"8b5d2e73e2a3a55c6c87b8b6eb92e0149a125c852751db1422fa951e42a09b82c142c3ea98d0d9930b056a3bc9896b8f", // bloXroute Max Profit
		"b0b07cd0abef743db4260b0ed50619cf6ad4d82064cb4fbec9d3ec530f7c5e6793d9f286c4e082c0244ffb9f2658fe88", // bloXroute Regulated
		"b3ee7afcf27f1f1259ac1787876318c6584ee353097a50ed84f51a1f21a323b3736f271a895c7ce918c038e4265918be", // Eden Network
		"ac6e77dfe25ecd6110b8e780608cce0dab71fdd5ebea22a16c0205200f2f8e2e3ad3b71d3499c54ad14d6c21b41a37ae", // Flashbots
		"98650451ba02064f7b000f5768cf0cf4d4e492317d82871bdc87ef841a0743f69f0f1eea11168503240ac35d101c9135", // Manifold
		"a1559ace749633b997cb3fdacffb890aeebdb0f5a3b6aaa7eeeaf1a38af0a8fe88b9e4b1f61f236d2e64d95733327a62", // Ultra Sound
		"8c7d33605ecef85403f8b7289c8058f440cbb6bf72b055dfe2f3e2c6695b6a1ea5a9cd0eb3a7982927a463feb4c3dae2", // Wenmerge
		"8c4ed5e24fe5c6ae21018437bde147693f68cda427cd1122cf20819c30eda7ed74f72dece09bb313f2a1855595ab677d", // TitanRelay
	}

	for _, tx := range transactions {
		for _, relay := range mevRelays {
			if strings.Contains(tx, relay) {
				return true
			}
		}
	}
	return false
}

// GetBlockReward retrieves the block reward status for a given slot
func GetBlockReward(c *gin.Context) {
	slot, err := strconv.Atoi(c.Param("slot"))
	if err != nil || slot < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid slot number"})
		return
	}

	blockData, err := FetchBlockData(slot)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch block data"})
		return
	}

	status := "Vanilla Block"
	if IsMEVBlock(blockData.Data.Message.Body.Transactions) {
		status = "MEV Relay"
	}

	response := BlockRewardResponse{
		Status: status,
	}

	c.JSON(http.StatusOK, response)
}
