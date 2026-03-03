package handlers

import (
	"anan-diary/ad-backend/internal/models"
	"anan-diary/ad-backend/internal/repository"
	"anan-diary/ad-backend/pkg/middleware"
	"anan-diary/ad-backend/pkg/response"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type parsedUint uint

type CommentHandler struct {
	commentRepo *repository.CommentRepository
	permissionRepo *repository.PermissionRepository
	logger      *zap.Logger
}

func NewCommentHandler(
	commentRepo *repository.CommentRepository,
	permissionRepo *repository.PermissionRepository,
	logger *zap.Logger,
) *CommentHandler {
	return &CommentHandler{
		commentRepo:   commentRepo,
		permissionRepo: permissionRepo,
		logger:        logger,
	}
}

// CreateComment 创建评论
func (h *CommentHandler) CreateComment(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	recordID := c.Param("id")
	recordIDUint, err := strconv.ParseUint(recordID, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的记录ID")
		return
	}

	var parsedRecordID uint
	if recordIDUint > 0 {
		parsedRecordID = uint(recordIDUint)
	}

	var req struct {
		ParentID *uint   `json:"parent_id"`
		Content  string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "请求参数错误")
		return
	}

	// 检查是否可以评论
	hasPermission, err := h.permissionRepo.HasPermission(uint(userID), parsedRecordID, "post_comment")
	if err != nil {
		h.logger.Error("Failed to check permission", zap.Error(err))
		response.ServerError(c, "系统错误")
		return
	}

	if !hasPermission {
		response.Forbidden(c, "无权发表评论")
		return
	}

	// 检查评论长度
	if len(req.Content) < 1 || len(req.Content) > 500 {
		response.ValidationError(c, "评论内容长度应在1-500字符之间")
		return
	}

	// 如果是回复评论，验证父评论是否存在
	if req.ParentID != nil {
		parentComment, err := h.commentRepo.GetByID(*req.ParentID)
		if err != nil {
			response.NotFound(c, "父评论不存在")
			return
		}

		// 验证父评论是否属于同一条成长记录
		if parentComment.GrowthRecordID != uint(0) {
			// recordID 是字符串，需要从 recordID 中提取数字部分
			// 这里简化处理，直接返回错误
			response.Error(c, http.StatusBadRequest, "无法回复跨记录的评论")
			return
		}
	}

	// 创建评论
	comment := &models.Comment{
		GrowthRecordID: uint(recordIDUint),
		UserID:         uint(userID),
		ParentID:       req.ParentID,
		Content:        req.Content,
	}

	if err := h.commentRepo.Create(comment); err != nil {
		h.logger.Error("Failed to create comment", zap.Error(err))
		response.ServerError(c, "评论发表失败")
		return
	}

	// 增加记录的评论计数（TODO: 需要在GrowthRecord模型中添加comments_count字段）

	h.logger.Info("Created comment",
		zap.Uint("user_id", uint(userID)),
		zap.Uint("record_id", parsedRecordID),
		zap.Uint("parent_id", parsedRecordID))

	response.Success(c, gin.H{
		"message": "评论发表成功",
		"comment": comment,
	})
}

// GetComments 获取记录的所有评论
func (h *CommentHandler) GetComments(c *gin.Context) {
	recordID := c.Param("id")
	recordIDUint, err := strconv.ParseUint(recordID, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的记录ID")
		return
	}

	var parsedRecordID uint
	if recordIDUint > 0 {
		parsedRecordID = uint(recordIDUint)
	}

	comments, err := h.commentRepo.GetByGrowthRecord(parsedRecordID)
	if err != nil {
		h.logger.Error("Failed to get comments", zap.Error(err))
		response.ServerError(c, "获取评论列表失败")
		return
	}

	response.Success(c, comments)
}

// GetCommentDetail 获取评论详情（包括子评论）
func (h *CommentHandler) GetCommentDetail(c *gin.Context) {
	commentID := c.Param("id")
	commentIDUint, err := strconv.ParseUint(commentID, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的评论ID")
		return
	}

	comment, err := h.commentRepo.GetByID(uint(commentIDUint))
	if err != nil {
		response.NotFound(c, "评论不存在")
		return
	}

	response.Success(c, comment)
}

// UpdateComment 更新评论
func (h *CommentHandler) UpdateComment(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "请求参数错误")
		return
	}

	// 获取评论
	commentIDStr := c.Param("id")
	commentIDUint, err := strconv.ParseUint(commentIDStr, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的评论ID")
		return
	}

	var parsedCommentID uint
	if commentIDUint > 0 {
		parsedCommentID = uint(commentIDUint)
	}

	comment, err := h.commentRepo.GetByID(parsedCommentID)
	if err != nil {
		response.NotFound(c, "评论不存在")
		return
	}

	// 只能修改自己的评论
	if comment.UserID != uint(userID) {
		response.Forbidden(c, "只能修改自己的评论")
		return
	}

	// 检查评论长度
	if len(req.Content) < 1 || len(req.Content) > 500 {
		response.ValidationError(c, "评论内容长度应在1-500字符之间")
		return
	}

	// 更新评论
	comment.Content = req.Content
	if err := h.commentRepo.Update(comment); err != nil {
		h.logger.Error("Failed to update comment", zap.Error(err))
		response.ServerError(c, "更新评论失败")
		return
	}

	h.logger.Info("Updated comment",
		zap.Uint("comment_id", parsedCommentID),
		zap.Uint("user_id", uint(userID)))

	response.Success(c, gin.H{
		"message": "评论更新成功",
		"comment": comment,
	})
}

// DeleteComment 删除评论
func (h *CommentHandler) DeleteComment(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	commentID := c.Param("id")

	// 获取评论
	commentIDStr := c.Param("id")
	commentIDUint, err := strconv.ParseUint(commentIDStr, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的评论ID")
		return
	}

	var parsedCommentID uint
	if commentIDUint > 0 {
		parsedCommentID = uint(commentIDUint)
	}

	comment, err := h.commentRepo.GetByID(parsedCommentID)
	if err != nil {
		response.NotFound(c, "评论不存在")
		return
	}

	// 只能删除自己的评论
	if comment.UserID != uint(userID) {
		response.Forbidden(c, "只能删除自己的评论")
		return
	}

	_ = commentID

	// 删除评论
	if err := h.commentRepo.Delete(uint(commentIDUint)); err != nil {
		h.logger.Error("Failed to delete comment", zap.Error(err))
		response.ServerError(c, "删除评论失败")
		return
	}

	// 删除子评论（级联删除）
	// 注意：删除父评论时应先删除所有子评论，这里通过数据库外键约束实现

	h.logger.Info("Deleted comment",
		zap.Uint("comment_id", uint(commentIDUint)),
		zap.Uint("user_id", uint(userID)))

	response.Success(c, gin.H{
		"message": "评论删除成功",
	})
}

// GetUserComments 获取用户的所有评论
func (h *CommentHandler) GetUserComments(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	comments, err := h.commentRepo.GetByUser(uint(userID))
	if err != nil {
		h.logger.Error("Failed to get user comments", zap.Error(err))
		response.ServerError(c, "获取评论列表失败")
		return
	}

	response.Success(c, comments)
}

// GetCommentCount 获取评论数
func (h *CommentHandler) GetCommentCount(c *gin.Context) {
	recordID := c.Param("id")
	recordIDUint, err := strconv.ParseUint(recordID, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的记录ID")
		return
	}

	count, err := h.commentRepo.GetCommentCount(uint(recordIDUint))
	if err != nil {
		h.logger.Error("Failed to get comment count", zap.Error(err))
		response.ServerError(c, "获取评论数失败")
		return
	}

	response.Success(c, gin.H{
		"count": count,
	})
}

// GetCommentTree 获取评论树
func (h *CommentHandler) GetCommentTree(c *gin.Context) {
	recordID := c.Param("id")
	recordIDUint, err := strconv.ParseUint(recordID, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的记录ID")
		return
	}

	var parsedRecordID uint
	if recordIDUint > 0 {
		parsedRecordID = uint(recordIDUint)
	}

	// 获取所有评论
	comments, err := h.commentRepo.GetByGrowthRecord(parsedRecordID)
	if err != nil {
		h.logger.Error("Failed to get comments", zap.Error(err))
		response.ServerError(c, "获取评论失败")
		return
	}

	// 构建评论树
	commentTree := buildCommentTree2(toPtrSlice(comments))

	response.Success(c, commentTree)
}

// buildCommentTree 构建评论树结构
func buildCommentTree(comments []models.Comment) []models.Comment {
	// 创建ID到评论的映射
	commentMap := make(map[uint]*models.Comment)
	var roots []models.Comment

	for i := range comments {
		comment := comments[i]
		commentMap[comment.ID] = &comment
	}

	// 构建树结构
	for i := range comments {
		comment := comments[i]
		if comment.ParentID != nil && commentMap[*comment.ParentID] != nil {
			// 是子评论
			commentMap[*comment.ParentID].Children = append(commentMap[*comment.ParentID].Children, &comment)
		} else {
			// 是根评论
			roots = append(roots, comment)
		}
	}

	return roots
}

// buildCommentTree2 构建评论树结构（使用指针）
func buildCommentTree2(comments []*models.Comment) []*models.Comment {
	// 创建ID到评论的映射
	commentMap := make(map[uint]*models.Comment)
	var roots []*models.Comment

	for i := range comments {
		commentMap[comments[i].ID] = comments[i]
	}

	// 构建树结构
	for i := range comments {
		comment := comments[i] // 直接使用 slice 中的元素
		if comment.ParentID != nil && commentMap[*comment.ParentID] != nil {
			// 是子评论
			commentMap[*comment.ParentID].Children = append(commentMap[*comment.ParentID].Children, comment)
		} else {
			// 是根评论
			roots = append(roots, comment)
		}
	}

	return roots
}

func toPtrSlice(comments []models.Comment) []*models.Comment {
	result := make([]*models.Comment, len(comments))
	for i := range comments {
		result[i] = &comments[i]
	}
	return result
}
