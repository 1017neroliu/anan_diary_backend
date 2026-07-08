package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	BabyID   string `json:"baby_id"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

const baseURL = "http://localhost:8080"

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║          宝宝时光轴 - 家庭成员测试脚本                  ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")

	// 模拟场景
	testRegisterMomAndCreateBaby()
	testRegisterDadAndLinkBaby()
	testDadLoginAndCancel()
	testDadAddRecord()
	testDadCreateBabyB()
	testDadAddRecordB()
	testMomLoginAndViewAll()

	fmt.Println("\n╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║                    所有测试场景完成                      ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
}

func testRegisterMomAndCreateBaby() {
	fmt.Println("\n【场景1】妈妈注册，创建宝宝A的档案")
	fmt.Println("───────────────────────────────────────────────────────────")

	// 妈妈注册
	mom := RegisterRequest{
		Email:    "mom@example.com",
		Password: "mom123",
		Name:     "李妈妈",
		Role:     "妈妈",
		BabyID:   "BABY001",
	}

	result := postRequest("/api/auth/register", mom)
	if result.Code != 200 {
		fmt.Printf("✗ 失败: %s\n", result.Message)
		return
	}

	data := result.Data.(map[string]interface{})
	babyID := int(data["baby_id"].(float64))
	fmt.Printf("✓ 妈妈注册成功 (mom@example.com)\n")
	fmt.Printf("✓ 宝宝档案创建成功 (baby_id=%d, baby_id=BABY001)\n", babyID)
	fmt.Printf("  ├─ 用户名: %s\n", data["name"])
	fmt.Printf("  ├─ 角色: %s\n", data["role"])
	fmt.Printf("  └─ 注册信息: 已关联宝宝档案\n")
}

func testRegisterDadAndLinkBaby() {
	fmt.Println("\n【场景2】爸爸注册，关联妈妈的宝宝")
	fmt.Println("───────────────────────────────────────────────────────────")

	// 爸爸注册
	dad := RegisterRequest{
		Email:    "dad@example.com",
		Password: "dad123",
		Name:     "张爸爸",
		Role:     "爸爸",
		BabyID:   "BABY001", // 使用相同的出生证编号
	}

	result := postRequest("/api/auth/register", dad)
	if result.Code != 200 {
		fmt.Printf("✗ 失败: %s\n", result.Message)
		return
	}

	data := result.Data.(map[string]interface{})
	babyID := int(data["baby_id"].(float64))
	fmt.Printf("✓ 爸爸注册成功 (dad@example.com)\n")
	fmt.Printf("✓ 成功关联妈妈的宝宝档案 (baby_id=%d)\n", babyID)
	fmt.Printf("  ├─ 用户名: %s\n", data["name"])
	fmt.Printf("  └─ 注册信息: 自动关联已存在的宝宝档案\n")
}

func testDadLoginAndCancel() {
	fmt.Println("\n【场景3】爸爸登录后点击取消，展示妈妈的记录")
	fmt.Println("───────────────────────────────────────────────────────────")

	// 爸爸登录
	login := LoginRequest{
		Email:    "dad@example.com",
		Password: "dad123",
	}

	result := postRequest("/api/auth/login", login)
	if result.Code != 200 {
		fmt.Printf("✗ 失败: %s\n", result.Message)
		return
	}

	babyID := result.Data.(map[string]interface{})["baby_id"]
	fmt.Printf("✓ 爸爸登录成功 (dad@example.com)\n")
	fmt.Printf("✓ 登录返回: baby_id=%v\n", babyID)
	fmt.Printf("  ├─ 预期: 点击取消后展示宝宝A的所有记录\n")
	fmt.Printf("  └─ 说明: baby_id已存在，无需创建新档案\n")

	// 获取时光轴数据
	timeline := getTimeline(babyID)
	if timeline.Code == 200 {
		records := timeline.Data.([]interface{})
		fmt.Printf("✓ 时光轴数据获取成功\n")
		fmt.Printf("  ├─ 记录数量: %d条\n", len(records))
		for i, r := range records {
			rec := r.(map[string]interface{})
			fmt.Printf("  └─ [%d] %s - %s (%s)\n", i+1,
				rec["created_at"],
				rec["content"],
				rec["member_role"])
		}
	}
}

func testDadAddRecord() {
	fmt.Println("\n【场景4】爸爸新增一条宝宝A的记录")
	fmt.Println("───────────────────────────────────────────────────────────")

	// 爸爸登录获取token
	login := LoginRequest{
		Email:    "dad@example.com",
		Password: "dad123",
	}
	dadResp := postRequest("/api/auth/login", login)
	dadToken := dadResp.Data.(map[string]interface{})["token"].(string)
	babyID := dadResp.Data.(map[string]interface{})["baby_id"]

	// 爸爸新增记录
	record := map[string]interface{}{
		"baby_id": babyID,
		"content": fmt.Sprintf("爸爸记录 - %s", time.Now().Format("2006-01-02")),
		"tags":    []string{"爸爸记录"},
	}

	result := postRequestWithToken("/api/growth-record", dadToken, record)
	if result.Code == 200 {
		fmt.Printf("✓ 爸爸新增记录成功\n")
		fmt.Printf("  ├─ 内容: %s\n", record["content"])
		fmt.Printf("  └─ 标签: 爸爸记录\n")
	}

	// 验证时光轴（应该有2条记录：妈妈的1条 + 爸爸的1条）
	timeline := getTimeline(babyID)
	if timeline.Code == 200 {
		records := timeline.Data.([]interface{})
		fmt.Printf("✓ 时光轴记录更新成功 (共%d条)\n", len(records))
		for i, r := range records {
			rec := r.(map[string]interface{})
			fmt.Printf("  ├─ [%d] %s - %s (%s)\n", i+1,
				rec["created_at"],
				rec["content"],
				rec["member_role"])
		}
	}
}

func testDadCreateBabyB() {
	fmt.Println("\n【场景5】爸爸创建宝宝B的档案")
	fmt.Println("───────────────────────────────────────────────────────────")

	// 爸爸注册（创建宝宝B）
	babyB := RegisterRequest{
		Email:    "dad@example.com", // 同一个账号
		Password: "dad123",
		Name:     "宝宝B",
		Role:     "爸爸",
		BabyID:   "BABY002",
	}

	result := postRequest("/api/auth/register", babyB)
	if result.Code != 200 {
		fmt.Printf("✗ 失败: %s\n", result.Message)
		return
	}

	data := result.Data.(map[string]interface{})
	babyBID := int(data["baby_id"].(float64))
	fmt.Printf("✓ 宝宝B档案创建成功\n")
	fmt.Printf("  ├─ 用户: dad@example.com (爸爸)\n")
	fmt.Printf("  ├─ 宝宝ID: %d\n", babyBID)
	fmt.Printf("  └─ 出生证编号: BABY002\n")

	// 验证爸爸的宝宝列表（应该有2个宝宝）
	babies := getBabies()
	if babies.Code == 200 {
		babyList := babies.Data.([]interface{})
		if len(babyList) == 2 {
			fmt.Printf("✓ 爸爸可以看到宝宝列表 (共%d个宝宝)\n", len(babyList))
			for i, b := range babyList {
				baby := b.(map[string]interface{})
				fmt.Printf("  ├─ [%d] baby_id=%d, name=%s\n", i+1,
					baby["id"], baby["name"])
			}
		}
	}
}

func testDadAddRecordB() {
	fmt.Println("\n【场景6】爸爸新增一条宝宝B的记录")
	fmt.Println("───────────────────────────────────────────────────────────")

	// 获取爸爸的token
	login := LoginRequest{
		Email:    "dad@example.com",
		Password: "dad123",
	}
	dadResp := postRequest("/api/auth/login", login)
	babyID := dadResp.Data.(map[string]interface{})["baby_id"]

	// 爸爸新增宝宝B的记录
	record := map[string]interface{}{
		"baby_id": babyID,
		"content": fmt.Sprintf("爸爸新增宝宝B的记录 - %s", time.Now().Format("2006-01-02")),
		"tags":    []string{"宝宝B"},
	}

	result := postRequestWithToken("/api/growth-record", dadResp.Data.(map[string]interface{})["token"].(string), record)
	if result.Code == 200 {
		fmt.Printf("✓ 爸爸新增宝宝B记录成功\n")
		fmt.Printf("  ├─ 内容: %s\n", record["content"])
		fmt.Printf("  └─ 标签: 宝宝B\n")
	}
}

func testMomLoginAndViewAll() {
	fmt.Println("\n【场景7】妈妈登录后查看宝宝A的记录和宝宝B的记录")
	fmt.Println("───────────────────────────────────────────────────────────")

	// 妈妈登录
	login := LoginRequest{
		Email:    "mom@example.com",
		Password: "mom123",
	}

	result := postRequest("/api/auth/login", login)
	if result.Code != 200 {
		fmt.Printf("✗ 失败: %s\n", result.Message)
		return
	}

	babyID := result.Data.(map[string]interface{})["baby_id"]
	fmt.Printf("✓ 妈妈登录成功 (mom@example.com)\n")
	fmt.Printf("✓ 登录返回: baby_id=%v\n", babyID)
	fmt.Printf("  ├─ 说明: 妈妈登录时返回的baby_id可能不同（取决于API实现）\n")
	fmt.Printf("  └─ 预期行为: 可以看到关联的宝宝列表\n")

	// 获取宝宝列表
	babies := getBabies()
	if babies.Code == 200 {
		babyList := babies.Data.([]interface{})
		if len(babyList) == 1 {
			fmt.Printf("✓ 妈妈可以看到宝宝列表 (共%d个宝宝)\n", len(babyList))
			for i, b := range babyList {
				baby := b.(map[string]interface{})
				fmt.Printf("  ├─ [%d] baby_id=%d, name=%s\n", i+1,
					baby["id"], baby["name"])
			}
		} else {
			fmt.Printf("  (返回了%d个宝宝)\n", len(babyList))
		}
	}

	// 获取时光轴数据（如果有baby_id）
	if babyID != nil {
		timeline := getTimeline(babyID)
		if timeline.Code == 200 {
			records := timeline.Data.([]interface{})
			fmt.Printf("✓ 时光轴数据获取成功 (共%d条记录)\n", len(records))
			for i, r := range records {
				rec := r.(map[string]interface{})
				fmt.Printf("  ├─ [%d] %s - %s (%s)\n", i+1,
					rec["created_at"],
					rec["content"],
					rec["member_role"])
			}
		}
	}
}

// ========== HTTP 请求辅助函数 ==========

func postRequest(path string, data interface{}) APIResponse {
	jsonData, _ := json.Marshal(data)
	resp, err := http.Post(baseURL+path, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return APIResponse{Code: 500, Message: err.Error()}
	}
	defer resp.Body.Close()

	var result APIResponse
	json.NewDecoder(resp.Body).Decode(&result)
	return result
}

func postRequestWithToken(path string, token string, data interface{}) APIResponse {
	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", baseURL+path, bytes.NewBuffer(jsonData))
	if err != nil {
		return APIResponse{Code: 500, Message: err.Error()}
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return APIResponse{Code: 500, Message: err.Error()}
	}
	defer resp.Body.Close()

	var result APIResponse
	json.NewDecoder(resp.Body).Decode(&result)
	return result
}

func getTimeline(babyID interface{}) APIResponse {
	path := fmt.Sprintf("/api/growth-record/timeline?baby_id=%v", babyID)
	resp, err := http.Get(baseURL + path)
	if err != nil {
		return APIResponse{Code: 500, Message: err.Error()}
	}
	defer resp.Body.Close()

	var result APIResponse
	json.NewDecoder(resp.Body).Decode(&result)
	return result
}

func getBabies() APIResponse {
	resp, err := http.Get(baseURL + "/api/baby")
	if err != nil {
		return APIResponse{Code: 500, Message: err.Error()}
	}
	defer resp.Body.Close()

	var result APIResponse
	json.NewDecoder(resp.Body).Decode(&result)
	return result
}

func readBody(resp *http.Response) string {
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return string(body)
}
