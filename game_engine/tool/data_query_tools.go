package tool

import (
	"context"
	"fmt"

	"github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"
)

// ========== 数据查询工具 ==========
// 所有数据查询工具均为只读，封装引擎的 List/Get 查询API

// ---------- 种族查询 ----------

// ListRacesTool 列出种族
type ListRacesTool struct {
	EngineTool
}

func NewListRacesTool(e *engine.Engine) *ListRacesTool {
	return &ListRacesTool{
		EngineTool: *NewEngineTool(
			"list_races",
			"列出所有可用的 D&D 5e 种族及其特性（属性加成、种族能力等）。Use when: 玩家询问有哪些种族可选；创建角色时需要参考种族列表。Do NOT use when: 需要某个种族的详细信息（用 get_race）；需要查询职业、背景等其他静态数据（用 list_classes/list_backgrounds 等对应工具）。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"page":      map[string]any{"type": "integer", "description": "页码（默认1）"},
					"page_size": map[string]any{"type": "integer", "description": "每页数量（默认20）"},
				},
			},
			e,
			true,
		),
	}
}

func (t *ListRacesTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	req := engine.ListRacesRequest{}
	if page := OptionalInt(params, "page", 0); page > 0 {
		pageSize := OptionalInt(params, "page_size", 20)
		req.Pagination = &engine.PaginationRequest{Page: page, PageSize: pageSize}
	}

	result, err := e.ListRaces(ctx, req)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: fmt.Sprintf("共 %d 个种族", result.Pagination.TotalCount),
	}, nil
}

// GetRaceTool 获取种族详情
type GetRaceTool struct {
	EngineTool
}

func NewGetRaceTool(e *engine.Engine) *GetRaceTool {
	return &GetRaceTool{
		EngineTool: *NewEngineTool(
			"get_race",
			"获取指定种族的完整信息，包括属性加成、种族特性、速度等。Use when: 玩家询问某个种族的具体能力；创建角色时需要确认种族的属性加成。参数 name 必须使用中文标准名（人类、精灵、矮人、半身人、龙裔、侏儒、半精灵、半兽人、提夫林）。Do NOT use when: 只需要种族列表（用 list_races）；需要查询的是职业或背景数据。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string", "description": "种族名称，必须使用中文标准名（如：人类、精灵、矮人、半身人、龙裔、侏儒、半精灵、半兽人、提夫林）"},
				},
				"required": []string{"name"},
			},
			e,
			true,
		),
	}
}

func (t *GetRaceTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	name, err := RequireString(params, "name")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	req := engine.GetRaceRequest{Name: name}
	result, err := e.GetRace(ctx, req)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	return &ToolResult{Success: true, Data: result, Message: result.Race.Name}, nil
}

// ---------- 职业查询 ----------

// ListClassesTool 列出职业
type ListClassesTool struct {
	EngineTool
}

func NewListClassesTool(e *engine.Engine) *ListClassesTool {
	return &ListClassesTool{
		EngineTool: *NewEngineTool(
			"list_classes",
			"列出所有可用的D&D职业",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"page":      map[string]any{"type": "integer", "description": "页码（默认1）"},
					"page_size": map[string]any{"type": "integer", "description": "每页数量（默认20）"},
				},
			},
			e,
			true,
		),
	}
}

func (t *ListClassesTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	req := engine.ListClassesRequest{}
	if page := OptionalInt(params, "page", 0); page > 0 {
		pageSize := OptionalInt(params, "page_size", 20)
		req.Pagination = &engine.PaginationRequest{Page: page, PageSize: pageSize}
	}

	result, err := e.ListClasses(ctx, req)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: fmt.Sprintf("共 %d 个职业", result.Pagination.TotalCount),
	}, nil
}

// GetClassTool 获取职业详情
type GetClassTool struct {
	EngineTool
}

func NewGetClassTool(e *engine.Engine) *GetClassTool {
	return &GetClassTool{
		EngineTool: *NewEngineTool(
			"get_class",
			"获取指定职业的详细信息",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string", "description": "职业ID"},
				},
				"required": []string{"id"},
			},
			e,
			true,
		),
	}
}

func (t *GetClassTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	idStr, err := RequireString(params, "id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	req := engine.GetClassRequest{ID: model.ClassID(idStr)}
	result, err := e.GetClass(ctx, req)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	return &ToolResult{Success: true, Data: result, Message: result.Class.Name}, nil
}

// ---------- 背景查询 ----------

// ListBackgroundsTool 列出背景
type ListBackgroundsTool struct {
	EngineTool
}

func NewListBackgroundsTool(e *engine.Engine) *ListBackgroundsTool {
	return &ListBackgroundsTool{
		EngineTool: *NewEngineTool(
			"list_backgrounds",
			"列出所有可用的角色背景",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"page":      map[string]any{"type": "integer", "description": "页码（默认1）"},
					"page_size": map[string]any{"type": "integer", "description": "每页数量（默认20）"},
				},
			},
			e,
			true,
		),
	}
}

func (t *ListBackgroundsTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	req := engine.ListBackgroundsRequest{}
	if page := OptionalInt(params, "page", 0); page > 0 {
		pageSize := OptionalInt(params, "page_size", 20)
		req.Pagination = &engine.PaginationRequest{Page: page, PageSize: pageSize}
	}

	result, err := e.ListBackgrounds(ctx, req)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: fmt.Sprintf("共 %d 个背景", result.Pagination.TotalCount),
	}, nil
}

// GetBackgroundTool 获取背景详情
type GetBackgroundTool struct {
	EngineTool
}

func NewGetBackgroundTool(e *engine.Engine) *GetBackgroundTool {
	return &GetBackgroundTool{
		EngineTool: *NewEngineTool(
			"get_background",
			"获取指定背景的详细信息",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string", "description": "背景ID或中文名称（如：侍僧、罪犯、学者、士兵）"},
				},
				"required": []string{"id"},
			},
			e,
			true,
		),
	}
}

func (t *GetBackgroundTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	id, err := RequireString(params, "id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	req := engine.GetBackgroundRequest{ID: id}
	result, err := e.GetBackground(ctx, req)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	return &ToolResult{Success: true, Data: result, Message: result.Background.Name}, nil
}

// ---------- 怪物查询 ----------

// ListMonstersTool 列出怪物
type ListMonstersTool struct {
	EngineTool
}

func NewListMonstersTool(e *engine.Engine) *ListMonstersTool {
	return &ListMonstersTool{
		EngineTool: *NewEngineTool(
			"list_monsters",
			"列出所有怪物数据",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"page":      map[string]any{"type": "integer", "description": "页码（默认1）"},
					"page_size": map[string]any{"type": "integer", "description": "每页数量（默认20）"},
				},
			},
			e,
			true,
		),
	}
}

func (t *ListMonstersTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	req := engine.ListMonstersRequest{}
	if page := OptionalInt(params, "page", 0); page > 0 {
		pageSize := OptionalInt(params, "page_size", 20)
		req.Pagination = &engine.PaginationRequest{Page: page, PageSize: pageSize}
	}

	result, err := e.ListMonsters(ctx, req)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: fmt.Sprintf("共 %d 个怪物", result.Pagination.TotalCount),
	}, nil
}

// GetMonsterTool 获取怪物详情
type GetMonsterTool struct {
	EngineTool
}

func NewGetMonsterTool(e *engine.Engine) *GetMonsterTool {
	return &GetMonsterTool{
		EngineTool: *NewEngineTool(
			"get_monster",
			"获取指定怪物的详细信息",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string", "description": "怪物ID"},
				},
				"required": []string{"id"},
			},
			e,
			true,
		),
	}
}

func (t *GetMonsterTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	id, err := RequireString(params, "id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	req := engine.GetMonsterRequest{ID: id}
	result, err := e.GetMonster(ctx, req)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	return &ToolResult{Success: true, Data: result, Message: result.Monster.Name}, nil
}

// ---------- 法术查询 ----------

// ListSpellsTool 列出法术
type ListSpellsTool struct {
	EngineTool
}

func NewListSpellsTool(e *engine.Engine) *ListSpellsTool {
	return &ListSpellsTool{
		EngineTool: *NewEngineTool(
			"list_spells",
			"列出所有可用的 D&D 5e 法术及其基本信息（环级、学派、施法时间等）。Use when: 玩家询问有哪些法术可选；施法者需要浏览可用法术列表。Do NOT use when: 需要某个法术的详细信息（用 get_spell）；需要施放法术（用 cast_spell，这是 write 操作）；需要查看施法者的法术位状态（用 get_spell_slots）。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"page":      map[string]any{"type": "integer", "description": "页码（默认1）"},
					"page_size": map[string]any{"type": "integer", "description": "每页数量（默认20）"},
				},
			},
			e,
			true,
		),
	}
}

func (t *ListSpellsTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	req := engine.ListSpellsRequest{}
	if page := OptionalInt(params, "page", 0); page > 0 {
		pageSize := OptionalInt(params, "page_size", 20)
		req.Pagination = &engine.PaginationRequest{Page: page, PageSize: pageSize}
	}

	result, err := e.ListSpells(ctx, req)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: fmt.Sprintf("共 %d 个法术", result.Pagination.TotalCount),
	}, nil
}

// GetSpellTool 获取法术详情
type GetSpellTool struct {
	EngineTool
}

func NewGetSpellTool(e *engine.Engine) *GetSpellTool {
	return &GetSpellTool{
		EngineTool: *NewEngineTool(
			"get_spell",
			"获取指定法术的完整信息，包括环级、学派、施法时间、射程、成分、持续时间、效果描述等。Use when: 玩家询问某个法术的具体效果；需要确认法术的伤害类型、豁免属性或作用范围。Do NOT use when: 需要施放该法术（用 cast_spell）；只需要法术列表概览（用 list_spells）。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string", "description": "法术ID"},
				},
				"required": []string{"id"},
			},
			e,
			true,
		),
	}
}

func (t *GetSpellTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	id, err := RequireString(params, "id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	req := engine.GetSpellRequest{ID: id}
	result, err := e.GetSpell(ctx, req)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	return &ToolResult{Success: true, Data: result, Message: result.Spell.Name}, nil
}

// ---------- 武器查询 ----------

// ListWeaponsTool 列出武器
type ListWeaponsTool struct {
	EngineTool
}

func NewListWeaponsTool(e *engine.Engine) *ListWeaponsTool {
	return &ListWeaponsTool{
		EngineTool: *NewEngineTool(
			"list_weapons",
			"列出所有武器数据",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"page":      map[string]any{"type": "integer", "description": "页码（默认1）"},
					"page_size": map[string]any{"type": "integer", "description": "每页数量（默认20）"},
				},
			},
			e,
			true,
		),
	}
}

func (t *ListWeaponsTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	req := engine.ListWeaponsRequest{}
	if page := OptionalInt(params, "page", 0); page > 0 {
		pageSize := OptionalInt(params, "page_size", 20)
		req.Pagination = &engine.PaginationRequest{Page: page, PageSize: pageSize}
	}

	result, err := e.ListWeapons(ctx, req)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: fmt.Sprintf("共 %d 种武器", result.Pagination.TotalCount),
	}, nil
}

// ---------- 护甲查询 ----------

// ListArmorsTool 列出护甲
type ListArmorsTool struct {
	EngineTool
}

func NewListArmorsTool(e *engine.Engine) *ListArmorsTool {
	return &ListArmorsTool{
		EngineTool: *NewEngineTool(
			"list_armors",
			"列出所有护甲数据",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"page":      map[string]any{"type": "integer", "description": "页码（默认1）"},
					"page_size": map[string]any{"type": "integer", "description": "每页数量（默认20）"},
				},
			},
			e,
			true,
		),
	}
}

func (t *ListArmorsTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	req := engine.ListArmorsRequest{}
	if page := OptionalInt(params, "page", 0); page > 0 {
		pageSize := OptionalInt(params, "page_size", 20)
		req.Pagination = &engine.PaginationRequest{Page: page, PageSize: pageSize}
	}

	result, err := e.ListArmors(ctx, req)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: fmt.Sprintf("共 %d 种护甲", result.Pagination.TotalCount),
	}, nil
}

// ---------- 魔法物品查询 ----------

// ListMagicItemsTool 列出魔法物品
type ListMagicItemsTool struct {
	EngineTool
}

func NewListMagicItemsTool(e *engine.Engine) *ListMagicItemsTool {
	return &ListMagicItemsTool{
		EngineTool: *NewEngineTool(
			"list_magic_items",
			"列出所有魔法物品数据",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"page":      map[string]any{"type": "integer", "description": "页码（默认1）"},
					"page_size": map[string]any{"type": "integer", "description": "每页数量（默认20）"},
				},
			},
			e,
			true,
		),
	}
}

func (t *ListMagicItemsTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	req := engine.ListMagicItemsRequest{}
	if page := OptionalInt(params, "page", 0); page > 0 {
		pageSize := OptionalInt(params, "page_size", 20)
		req.Pagination = &engine.PaginationRequest{Page: page, PageSize: pageSize}
	}

	result, err := e.ListMagicItems(ctx, req)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: fmt.Sprintf("共 %d 个魔法物品", result.Pagination.TotalCount),
	}, nil
}

// ---------- 专长查询 ----------

// ListFeatsDataTool 列出专长
type ListFeatsDataTool struct {
	EngineTool
}

func NewListFeatsDataTool(e *engine.Engine) *ListFeatsDataTool {
	return &ListFeatsDataTool{
		EngineTool: *NewEngineTool(
			"list_feats_data",
			"列出所有专长数据",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"page":      map[string]any{"type": "integer", "description": "页码（默认1）"},
					"page_size": map[string]any{"type": "integer", "description": "每页数量（默认20）"},
				},
			},
			e,
			true,
		),
	}
}

func (t *ListFeatsDataTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	req := engine.ListFeatsDataRequest{}
	if page := OptionalInt(params, "page", 0); page > 0 {
		pageSize := OptionalInt(params, "page_size", 20)
		req.Pagination = &engine.PaginationRequest{Page: page, PageSize: pageSize}
	}

	result, err := e.ListFeatsData(ctx, req)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Message: fmt.Sprintf("共 %d 个专长", result.Pagination.TotalCount),
	}, nil
}

// GetFeatDataTool 获取专长详情
type GetFeatDataTool struct {
	EngineTool
}

func NewGetFeatDataTool(e *engine.Engine) *GetFeatDataTool {
	return &GetFeatDataTool{
		EngineTool: *NewEngineTool(
			"get_feat_data",
			"获取指定专长的详细信息",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string", "description": "专长ID"},
				},
				"required": []string{"id"},
			},
			e,
			true,
		),
	}
}

func (t *GetFeatDataTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	id, err := RequireString(params, "id")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	req := engine.GetFeatDataRequest{ID: id}
	result, err := e.GetFeatData(ctx, req)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	return &ToolResult{Success: true, Data: result, Message: result.Feat.Name}, nil
}
