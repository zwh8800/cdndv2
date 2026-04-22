package tool

import (
	"context"
	"fmt"

	"github.com/zwh8800/dnd-core/pkg/engine"
	"github.com/zwh8800/dnd-core/pkg/model"
)

// =============================================================================
// 22. lookup_game_data - 统一D&D规则数据查询
// =============================================================================

type LookupGameDataTool struct {
	EngineTool
}

func NewLookupGameDataTool(e *engine.Engine) *LookupGameDataTool {
	return &LookupGameDataTool{
		EngineTool: *NewEngineTool(
			"lookup_game_data",
			"统一D&D规则数据查询。查询种族、职业、背景、怪物、法术、武器、盔甲、魔法物品、专长等游戏数据",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"data_type": map[string]any{
						"type": "string",
						"enum": []string{"race", "class", "background", "monster", "spell", "weapon", "armor", "magic_item", "feat"},
						"description": "数据类型",
					},
					"name": map[string]any{
						"type":        "string",
						"description": "查询单个详情时的名称/ID。不提供则返回列表",
					},
					"page":      map[string]any{"type": "integer", "description": "页码（列表查询时可选，默认1）"},
					"page_size": map[string]any{"type": "integer", "description": "每页数量（默认20）"},
				},
				"required": []string{"data_type"},
			},
			e,
			true, // 只读
		),
	}
}

func (t *LookupGameDataTool) Execute(ctx context.Context, params map[string]any) (*ToolResult, error) {
	e := t.Engine().(*engine.Engine)

	dataType, err := RequireString(params, "data_type")
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	name := OptionalString(params, "name", "")

	// 构建分页参数
	var pagination *engine.PaginationRequest
	if page := OptionalInt(params, "page", 0); page > 0 {
		pagination = &engine.PaginationRequest{Page: page, PageSize: OptionalInt(params, "page_size", 20)}
	}

	switch dataType {
	case "race":
		if name != "" {
			result, qerr := e.GetRace(ctx, engine.GetRaceRequest{Name: name})
			if qerr != nil {
				return &ToolResult{Success: false, Error: qerr.Error()}, nil
			}
			return &ToolResult{Success: true, Data: result, Message: result.Race.Name}, nil
		}
		result, qerr := e.ListRaces(ctx, engine.ListRacesRequest{Pagination: pagination})
		if qerr != nil {
			return &ToolResult{Success: false, Error: qerr.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: fmt.Sprintf("共 %d 个种族", result.Pagination.TotalCount)}, nil

	case "class":
		if name != "" {
			result, qerr := e.GetClass(ctx, engine.GetClassRequest{ID: model.ClassID(name)})
			if qerr != nil {
				return &ToolResult{Success: false, Error: qerr.Error()}, nil
			}
			return &ToolResult{Success: true, Data: result, Message: result.Class.Name}, nil
		}
		result, qerr := e.ListClasses(ctx, engine.ListClassesRequest{Pagination: pagination})
		if qerr != nil {
			return &ToolResult{Success: false, Error: qerr.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: fmt.Sprintf("共 %d 个职业", result.Pagination.TotalCount)}, nil

	case "background":
		if name != "" {
			result, qerr := e.GetBackground(ctx, engine.GetBackgroundRequest{ID: name})
			if qerr != nil {
				return &ToolResult{Success: false, Error: qerr.Error()}, nil
			}
			return &ToolResult{Success: true, Data: result, Message: result.Background.Name}, nil
		}
		result, qerr := e.ListBackgrounds(ctx, engine.ListBackgroundsRequest{Pagination: pagination})
		if qerr != nil {
			return &ToolResult{Success: false, Error: qerr.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: fmt.Sprintf("共 %d 个背景", result.Pagination.TotalCount)}, nil

	case "monster":
		if name != "" {
			result, qerr := e.GetMonster(ctx, engine.GetMonsterRequest{ID: name})
			if qerr != nil {
				return &ToolResult{Success: false, Error: qerr.Error()}, nil
			}
			return &ToolResult{Success: true, Data: result, Message: result.Monster.Name}, nil
		}
		result, qerr := e.ListMonsters(ctx, engine.ListMonstersRequest{Pagination: pagination})
		if qerr != nil {
			return &ToolResult{Success: false, Error: qerr.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: fmt.Sprintf("共 %d 个怪物", result.Pagination.TotalCount)}, nil

	case "spell":
		if name != "" {
			result, qerr := e.GetSpell(ctx, engine.GetSpellRequest{ID: name})
			if qerr != nil {
				return &ToolResult{Success: false, Error: qerr.Error()}, nil
			}
			return &ToolResult{Success: true, Data: result, Message: result.Spell.Name}, nil
		}
		result, qerr := e.ListSpells(ctx, engine.ListSpellsRequest{Pagination: pagination})
		if qerr != nil {
			return &ToolResult{Success: false, Error: qerr.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: fmt.Sprintf("共 %d 个法术", result.Pagination.TotalCount)}, nil

	case "weapon":
		result, qerr := e.ListWeapons(ctx, engine.ListWeaponsRequest{Pagination: pagination})
		if qerr != nil {
			return &ToolResult{Success: false, Error: qerr.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: fmt.Sprintf("共 %d 把武器", result.Pagination.TotalCount)}, nil

	case "armor":
		result, qerr := e.ListArmors(ctx, engine.ListArmorsRequest{Pagination: pagination})
		if qerr != nil {
			return &ToolResult{Success: false, Error: qerr.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: fmt.Sprintf("共 %d 件盔甲", result.Pagination.TotalCount)}, nil

	case "magic_item":
		result, qerr := e.ListMagicItems(ctx, engine.ListMagicItemsRequest{Pagination: pagination})
		if qerr != nil {
			return &ToolResult{Success: false, Error: qerr.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: fmt.Sprintf("共 %d 件魔法物品", result.Pagination.TotalCount)}, nil

	case "feat":
		if name != "" {
			result, qerr := e.GetFeatData(ctx, engine.GetFeatDataRequest{ID: name})
			if qerr != nil {
				return &ToolResult{Success: false, Error: qerr.Error()}, nil
			}
			return &ToolResult{Success: true, Data: result, Message: result.Feat.Name}, nil
		}
		result, qerr := e.ListFeatsData(ctx, engine.ListFeatsDataRequest{Pagination: pagination})
		if qerr != nil {
			return &ToolResult{Success: false, Error: qerr.Error()}, nil
		}
		return &ToolResult{Success: true, Data: result, Message: fmt.Sprintf("共 %d 个专长", result.Pagination.TotalCount)}, nil

	default:
		return &ToolResult{Success: false, Error: fmt.Sprintf("无效的data_type: %s", dataType)}, nil
	}
}
