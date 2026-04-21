package commands

import (
	"context"
	"fmt"

	"claudego/pkg/compaction"
	"claudego/pkg/logger"
	"claudego/pkg/types"
)

// HandleCompactCommand executes manual full compaction.
func HandleCompactCommand(ctx context.Context, compactor *compaction.Compactor, messages []types.Message, sessionID string, log *logger.Logger) ([]types.Message, error) {
	if compactor == nil {
		return nil, fmt.Errorf("compactor is nil")
	}
	if log != nil {
		log.Info("manual compaction triggered")
	}

	before := len(messages)
	compacted, err := compactor.FullCompact(ctx, messages, sessionID)
	if err != nil {
		return nil, fmt.Errorf("compaction failed: %w", err)
	}

	if log != nil {
		log.Info("compaction completed messages_before=%d messages_after=%d", before, len(compacted))
	}
	return compacted, nil
}

// HandleCompactStatusCommand logs the current compaction status.
func HandleCompactStatusCommand(compactor *compaction.Compactor, messages []types.Message, sessionID string, sessionTokens int, log *logger.Logger) error {
	if compactor == nil {
		return fmt.Errorf("compactor is nil")
	}
	totalTokens := compactor.TokenCounter().CountMessages(messages)
	threshold := compactor.Config().AutoCompactTokens
	percentage := 0
	if threshold > 0 {
		percentage = (totalTokens * 100) / threshold
	}

	if log != nil {
		log.Info(
			"compaction status session_id=%s messages=%d tokens=%d/%d (%d%%) session_total_tokens=%d",
			sessionID,
			len(messages),
			totalTokens,
			threshold,
			percentage,
			sessionTokens,
		)
	}
	return nil
}
