package handlers

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Constants for API base URL and endpoints
const (
	nodeURL             = "https://radial-misty-butterfly.quiknode.pro/d71f751e03f2b6466202f2561941b6c1c0defd13"
	blockEndpoint       = "/eth/v2/beacon/blocks/%d"
	validatorEndpoint   = "/eth/v1/beacon/states/head/validators/%d"
	validatorsEndpoint  = "/eth/v1/beacon/states/head/validators"
	BaseRewardFactor    = 64
	EthToGwei           = 1e9            // Conversion factor from ETH to Gwei
	MaxEffectiveBalance = 32 * EthToGwei // 32 ETH in Gwei
)

// BlockRewardResponse defines the structure of the response for the block reward endpoint
type BlockRewardResponse struct {
	Status string `json:"status"`
	Reward string `json:"reward"`
}

// BlockData represents the structure for block data returned by the beacon node
type BlockData struct {
	Data struct {
		Message struct {
			Slot          string `json:"slot"`
			ProposerIndex string `json:"proposer_index"`
			Body          struct {
				ExecutionPayload struct {
					FeeRecipient  string   `json:"fee_recipient"`
					BlockNumber   string   `json:"block_number"`
					GasLimit      string   `json:"gas_limit"`
					GasUsed       string   `json:"gas_used"`
					BaseFeePerGas string   `json:"base_fee_per_gas"`
					Transactions  []string `json:"transactions"`
					Withdrawals   []struct {
						Amount string `json:"amount"`
					} `json:"withdrawals"`
					ExtraData string `json:"extra_data"`
					LogsBloom string `json:"logs_bloom"`
				} `json:"execution_payload"`
			} `json:"body"`
		} `json:"message"`
	} `json:"data"`
}

// ValidatorResponse represents the structure for the validator's effective balance
type ValidatorResponse struct {
	Data struct {
		Validator struct {
			EffectiveBalance string `json:"effective_balance"`
		} `json:"validator"`
	} `json:"data"`
}

// ValidatorsResponse represents the structure for the total ETH staked in the network
type ValidatorsResponse struct {
	Data []struct {
		Validator struct {
			EffectiveBalance string `json:"effective_balance"`
		} `json:"validator"`
	} `json:"data"`
}

// RPCRequest represents the structure for the JSON-RPC request
type RPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

// RPCResponse represents the structure for the JSON-RPC response
type RPCResponse struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      int                    `json:"id"`
	Result  map[string]interface{} `json:"result"`
}

// Transaction represents a simplified transaction structure
type Transaction struct {
	GasPrice             string `json:"gasPrice"`
	MaxPriorityFeePerGas string `json:"maxPriorityFeePerGas"`
	GasUsed              string `json:"gas"`
}

// FetchBlockData retrieves block data for a given slot
func FetchBlockData(slot int) (*BlockData, error) {
	resp, err := http.Get(fmt.Sprintf(nodeURL+blockEndpoint, slot))
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

// FetchValidatorBalance fetches the effective balance of the validator
func FetchValidatorBalance(proposerIndex int) (float64, error) {
	url := fmt.Sprintf(nodeURL+validatorEndpoint, proposerIndex)
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var validator ValidatorResponse
	if err := json.NewDecoder(resp.Body).Decode(&validator); err != nil {
		return 0, err
	}

	effectiveBalance, err := strconv.ParseFloat(validator.Data.Validator.EffectiveBalance, 64)
	if err != nil {
		return 0, err
	}

	return effectiveBalance, nil
}

// FetchTotalStaked fetches the total ETH staked in the network
func FetchTotalStaked() (float64, error) {
	resp, err := http.Get(nodeURL + validatorsEndpoint)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var validatorsResp ValidatorsResponse
	if err := json.NewDecoder(resp.Body).Decode(&validatorsResp); err != nil {
		return 0, err
	}

	totalStaked := 0.0
	for _, validator := range validatorsResp.Data {
		balance, err := strconv.ParseFloat(validator.Validator.EffectiveBalance, 64)
		if err != nil {
			return 0, err
		}
		totalStaked += balance
	}

	return totalStaked, nil
}

// CalculateBaseReward calculates the base reward for a validator
func CalculateBaseReward(proposerIndex int) (float64, error) {
	// Fetch the effective balance of the validator
	effectiveBalance, err := FetchValidatorBalance(proposerIndex)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch validator balance: %v", err)
	}

	// Fetch the total staked ETH in the network
	totalStaked, err := FetchTotalStaked()
	if err != nil {
		return 0, fmt.Errorf("failed to fetch total staked: %v", err)
	}

	// Cap the effective balance at 32 ETH in Gwei
	if effectiveBalance > MaxEffectiveBalance {
		effectiveBalance = MaxEffectiveBalance
	}

	// Calculate the base reward
	baseReward := (BaseRewardFactor * effectiveBalance) / math.Sqrt(totalStaked)

	return baseReward, nil
}

// IsMEVBlock determines if a block was produced using an MEV relay by checking known MEV relay addresses
func IsMEVBlock(blockData *BlockData) bool {
	// List of known MEV relay identifiers (substrings) in lowercase
	knownRelays := []string{
		"aestus",
		"agnostic",  // Agnostic Gnosis
		"bloxroute", // Covers both bloxroute max profit and bloxroute regulated
		"eden",      // Eden Network
		"flashbots",
		"manifold",
		"ultra", // Ultra Sound
		"wenmerge",
		"titan", // TitanRelay
	}

	extraData := blockData.Data.Message.Body.ExecutionPayload.ExtraData
	decodedExtraData, err := hex.DecodeString(extraData[2:])
	if err != nil {
		return false
	}
	decodedString := strings.ToLower(string(decodedExtraData))

	// Check if the decoded extra_data contains any known MEV relay substring
	for _, relay := range knownRelays {
		if strings.Contains(decodedString, relay) {
			return true
		}
	}
	return false
}

// FetchBlockDetails retrieves block details including transactions
func FetchBlockDetails(blockNumber int64, quickNodeURL string) (*RPCResponse, error) {
	hexBlockNumber := fmt.Sprintf("0x%x", blockNumber)
	rpcRequest := RPCRequest{
		JSONRPC: "2.0",
		Method:  "eth_getBlockByNumber",
		Params:  []interface{}{hexBlockNumber, true},
		ID:      1,
	}

	jsonData, err := json.Marshal(rpcRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RPC request: %v", err)
	}

	resp, err := http.Post(quickNodeURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to send RPC request: %v", err)
	}
	defer resp.Body.Close()

	var rpcResponse RPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResponse); err != nil {
		return nil, fmt.Errorf("failed to decode RPC response: %v", err)
	}

	return &rpcResponse, nil
}

// HexToFloat converts a hexadecimal string to a float64 value
func HexToFloat(hexStr string) (float64, error) {
	value, err := strconv.ParseUint(hexStr[2:], 16, 64)
	if err != nil {
		return 0, err
	}
	return float64(value), nil
}

// CalculateProposerPayment calculates the proposer payment from an MEV relay.
// This function sums up the payments made by transactions that have a maxPriorityFeePerGas
// higher than the base fee. This difference represents the additional incentive (MEV payment)
// that goes to the proposer. The assumption here is that transactions with a maxPriorityFeePerGas
// greater than the base fee are part of the MEV bundle and are paying an additional incentive
// to the block proposer. We calculate this by summing up the gas fees associated with these
// transactions, thereby estimating the total MEV payment included in the block.
func CalculateProposerPayment(blockDetails *RPCResponse) (float64, error) {
	var proposerPayment float64

	for _, tx := range blockDetails.Result["transactions"].([]interface{}) {
		transaction := tx.(map[string]interface{})

		// Check if maxPriorityFeePerGas exists
		maxPriorityFeePerGasStr, ok := transaction["maxPriorityFeePerGas"].(string)
		if !ok || maxPriorityFeePerGasStr == "" {
			continue
		}

		maxPriorityFeePerGas, err := HexToFloat(maxPriorityFeePerGasStr)
		if err != nil {
			return 0, fmt.Errorf("failed to parse maxPriorityFeePerGas: %v", err)
		}

		gasUsedStr, ok := transaction["gas"].(string)
		if !ok || gasUsedStr == "" {
			continue
		}

		gasUsed, err := HexToFloat(gasUsedStr)
		if err != nil {
			return 0, fmt.Errorf("failed to parse gas used: %v", err)
		}

		// Calculate the payment based on gas used and the max priority fee per gas
		payment := maxPriorityFeePerGas * gasUsed
		proposerPayment += payment
	}

	return proposerPayment, nil
}

// CalculateTransactionFees calculates the total transaction fees for a block
func CalculateTransactionFees(blockDetails *RPCResponse) (float64, error) {
	baseFee, err := HexToFloat(blockDetails.Result["baseFeePerGas"].(string))
	if err != nil {
		return 0, fmt.Errorf("failed to parse base fee per gas: %v", err)
	}

	var totalFees float64

	for _, tx := range blockDetails.Result["transactions"].([]interface{}) {
		transaction := tx.(map[string]interface{})

		gasUsed, err := HexToFloat(transaction["gas"].(string))
		if err != nil {
			return 0, fmt.Errorf("failed to parse gas used: %v", err)
		}

		gasPrice, err := HexToFloat(transaction["gasPrice"].(string))
		if err != nil {
			return 0, fmt.Errorf("failed to parse gas price: %v", err)
		}

		tip := gasPrice - baseFee

		totalFees += tip * gasUsed
	}

	return totalFees, nil
}

// GetBlockReward retrieves the block reward status for a given slot
func GetBlockReward(c *gin.Context) {
	// Parse slot parameter
	slot, err := strconv.Atoi(c.Param("slot"))
	if err != nil || slot < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid slot number"})
		return
	}

	// Fetch block data for the given slot
	blockData, err := FetchBlockData(slot)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch block data"})
		return
	}

	// Determine if the block was produced using an MEV relay
	status := "Vanilla Block"
	isMEV := IsMEVBlock(blockData)
	if isMEV {
		status = "MEV Relay"
	}

	// Extract proposer index and calculate base reward
	proposerIndex, err := strconv.Atoi(blockData.Data.Message.ProposerIndex)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid proposer index"})
		return
	}

	baseReward, err := CalculateBaseReward(proposerIndex)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to calculate base reward: %v", err)})
		return
	}

	// Initialize total reward with base reward
	totalReward := baseReward

	blockNumber, err := strconv.ParseInt(blockData.Data.Message.Body.ExecutionPayload.BlockNumber, 10, 64)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid block number"})
		return
	}

	// Fetch block details for the given block number
	blockDetails, err := FetchBlockDetails(blockNumber, nodeURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch block details: %v", err)})
		return
	}

	if isMEV {
		// For MEV blocks, calculate the proposer payment
		proposerPayment, err := CalculateProposerPayment(blockDetails)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to calculate proposer payment: %v", err)})
			return
		}
		totalReward += proposerPayment
	} else {
		// For vanilla blocks, calculate the transaction fees and add to total reward
		transactionFees, err := CalculateTransactionFees(blockDetails)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to calculate transaction fees: %v", err)})
			return
		}
		totalReward += transactionFees
	}

	// Convert total reward from Wei to Gwei
	finalReward := fmt.Sprintf("%.3f", totalReward/1e9)

	response := BlockRewardResponse{
		Status: status,
		Reward: finalReward,
	}

	c.JSON(http.StatusOK, response)
}
