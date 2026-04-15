package main

import (
	"bytes"
	"fmt"
	"html/template"
	"math"
	"os"
	"strings"
	"time"

	"gore/internal/advisor"
)

// severityClass returns CSS class for severity
func severityClass(sev advisor.Severity) string {
	switch sev {
	case advisor.SeverityInfo:
		return "info"
	case advisor.SeverityWarn:
		return "warning"
	case advisor.SeverityHigh:
		return "high"
	case advisor.SeverityCritical:
		return "critical"
	default:
		return "info"
	}
}

// severityLabel returns label for severity
func severityLabel(sev advisor.Severity) string {
	switch sev {
	case advisor.SeverityInfo:
		return "INFO"
	case advisor.SeverityWarn:
		return "WARNING"
	case advisor.SeverityHigh:
		return "HIGH"
	case advisor.SeverityCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// groupBySeverity groups suggestions by severity
func groupBySeverity(suggestions []advisor.Suggestion) map[advisor.Severity][]advisor.Suggestion {
	result := make(map[advisor.Severity][]advisor.Suggestion)
	for _, s := range suggestions {
		result[s.Severity] = append(result[s.Severity], s)
	}
	return result
}

// getRuleCounts returns count of each rule
func getRuleCounts(suggestions []advisor.Suggestion) map[string]int {
	counts := make(map[string]int)
	for _, s := range suggestions {
		counts[s.RuleID]++
	}
	return counts
}

// ruleSeverity returns severity for a rule ID
func ruleSeverity(ruleID string) advisor.Severity {
	sevMap := map[string]advisor.Severity{
		"IDX-001": advisor.SeverityWarn, "IDX-002": advisor.SeverityWarn,
		"IDX-003": advisor.SeverityWarn, "IDX-004": advisor.SeverityWarn,
		"IDX-005": advisor.SeverityWarn, "IDX-006": advisor.SeverityHigh,
		"IDX-007": advisor.SeverityInfo, "IDX-008": advisor.SeverityInfo,
		"IDX-009": advisor.SeverityWarn, "IDX-010": advisor.SeverityInfo,
	}
	if sev, ok := sevMap[ruleID]; ok {
		return sev
	}
	return advisor.SeverityInfo
}

// HTMLReport generates a beautiful HTML report with visualizations.
func HTMLReport(rep report, path string) error {
	chartSVG := generatePieChart(rep.Stats)
	ruleCounts := getRuleCounts(rep.Suggestions)

	data := map[string]any{
		"Report":      rep,
		"ChartSVG":    template.HTML(chartSVG),
		"RuleCounts":  ruleCounts,
		"GeneratedAt": time.Now().UTC().Format("2006-01-02 15:04:05 UTC"),
	}

	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>gore-lint Report</title>
	<link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600&family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
	<style>
		* { margin: 0; padding: 0; box-sizing: border-box; }

		body {
			font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
			background: linear-gradient(135deg, #0f0f23 0%, #1a1a2e 50%, #16213e 100%);
			min-height: 100vh;
			color: #e4e4e7;
			line-height: 1.6;
		}

		.container { max-width: 1400px; margin: 0 auto; padding: 40px 24px; }

		.header {
			display: flex;
			align-items: center;
			justify-content: space-between;
			margin-bottom: 40px;
			padding-bottom: 24px;
			border-bottom: 1px solid rgba(255,255,255,0.1);
		}

		.logo { display: flex; align-items: center; gap: 16px; }
		.logo-icon {
			width: 48px; height: 48px;
			background: linear-gradient(135deg, #6366f1, #8b5cf6);
			border-radius: 12px;
			display: flex; align-items: center; justify-content: center;
			font-size: 24px;
		}
		.logo h1 {
			font-size: 28px; font-weight: 700;
			background: linear-gradient(90deg, #6366f1, #a855f7, #ec4899);
			-webkit-background-clip: text; -webkit-text-fill-color: transparent;
		}
		.logo span { font-size: 14px; color: #71717a; }
		.meta { text-align: right; font-size: 13px; color: #71717a; }
		.meta strong { color: #a1a1aa; }

		.stats-grid {
			display: grid;
			grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
			gap: 20px;
			margin-bottom: 40px;
		}

		.stat-card {
			background: rgba(255,255,255,0.03);
			border: 1px solid rgba(255,255,255,0.08);
			border-radius: 16px;
			padding: 24px;
			position: relative;
			overflow: hidden;
			transition: transform 0.2s, box-shadow 0.2s;
		}
		.stat-card:hover { transform: translateY(-2px); box-shadow: 0 8px 30px rgba(0,0,0,0.3); }
		.stat-card::before { content: ''; position: absolute; top: 0; left: 0; right: 0; height: 3px; }
		.stat-card.total::before { background: linear-gradient(90deg, #6366f1, #8b5cf6); }
		.stat-card.info::before { background: linear-gradient(90deg, #22d3ee, #06b6d4); }
		.stat-card.warning::before { background: linear-gradient(90deg, #fbbf24, #f59e0b); }
		.stat-card.high::before { background: linear-gradient(90deg, #f87171, #ef4444); }
		.stat-card.critical::before { background: linear-gradient(90deg, #f472b6, #db2777); }

		.stat-label { font-size: 12px; color: #71717a; text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 8px; }
		.stat-value { font-size: 42px; font-weight: 700; font-family: 'JetBrains Mono', monospace; }
		.stat-card.total .stat-value { color: #a78bfa; }
		.stat-card.info .stat-value { color: #22d3ee; }
		.stat-card.warning .stat-value { color: #fbbf24; }
		.stat-card.high .stat-value { color: #f87171; }
		.stat-card.critical .stat-value { color: #f472b6; }

		.chart-section {
			display: grid;
			grid-template-columns: 1fr 1fr;
			gap: 24px;
			margin-bottom: 40px;
		}

		.chart-card {
			background: rgba(255,255,255,0.03);
			border: 1px solid rgba(255,255,255,0.08);
			border-radius: 16px;
			padding: 24px;
		}

		.chart-title {
			font-size: 16px; font-weight: 600; color: #e4e4e7;
			margin-bottom: 20px;
			display: flex; align-items: center; gap: 8px;
		}
		.chart-title::before { content: ''; width: 4px; height: 16px; background: linear-gradient(180deg, #6366f1, #a855f7); border-radius: 2px; }

		.pie-chart { display: flex; align-items: center; justify-content: center; gap: 40px; }
		.legend { display: flex; flex-direction: column; gap: 12px; }
		.legend-item { display: flex; align-items: center; gap: 10px; font-size: 14px; }
		.legend-dot { width: 12px; height: 12px; border-radius: 50%; }
		.legend-dot.info { background: #22d3ee; }
		.legend-dot.warning { background: #fbbf24; }
		.legend-dot.high { background: #f87171; }
		.legend-dot.critical { background: #f472b6; }
		.legend-label { color: #a1a1aa; }
		.legend-value { color: #e4e4e7; font-weight: 600; font-family: 'JetBrains Mono', monospace; }

		.rules-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(140px, 1fr)); gap: 12px; }
		.rule-badge { display: flex; align-items: center; gap: 8px; padding: 12px 16px; background: rgba(255,255,255,0.05); border-radius: 8px; font-size: 13px; }
		.rule-badge.info { border-left: 3px solid #22d3ee; }
		.rule-badge.warning { border-left: 3px solid #fbbf24; }
		.rule-badge.high { border-left: 3px solid #f87171; }
		.rule-badge.critical { border-left: 3px solid #f472b6; }

		.issues-section { margin-top: 40px; }
		.section-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 24px; }
		.section-title { font-size: 20px; font-weight: 600; display: flex; align-items: center; gap: 12px; }
		.section-title .count { background: rgba(255,255,255,0.1); padding: 4px 12px; border-radius: 20px; font-size: 14px; font-weight: 500; }

		.severity-group { margin-bottom: 32px; }
		.severity-header {
			display: flex; align-items: center; gap: 12px; padding: 12px 16px;
			background: rgba(255,255,255,0.02); border-radius: 8px; margin-bottom: 16px;
		}
		.severity-indicator { width: 8px; height: 8px; border-radius: 50%; }
		.severity-indicator.info { background: #22d3ee; box-shadow: 0 0 10px #22d3ee; }
		.severity-indicator.warning { background: #fbbf24; box-shadow: 0 0 10px #fbbf24; }
		.severity-indicator.high { background: #f87171; box-shadow: 0 0 10px #f87171; }
		.severity-indicator.critical { background: #f472b6; box-shadow: 0 0 10px #f472b6; }
		.severity-name { font-weight: 600; font-size: 14px; text-transform: uppercase; letter-spacing: 0.5px; }
		.severity-name.info { color: #22d3ee; }
		.severity-name.warning { color: #fbbf24; }
		.severity-name.high { color: #f87171; }
		.severity-name.critical { color: #f472b6; }
		.severity-count { background: rgba(255,255,255,0.1); padding: 2px 10px; border-radius: 10px; font-size: 12px; font-weight: 600; }

		.issue-card {
			background: rgba(255,255,255,0.03);
			border: 1px solid rgba(255,255,255,0.06);
			border-radius: 12px;
			padding: 20px;
			margin-bottom: 12px;
			transition: transform 0.2s, border-color 0.2s;
		}
		.issue-card:hover { transform: translateX(4px); border-color: rgba(255,255,255,0.12); }
		.issue-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; margin-bottom: 12px; }
		.issue-rule { display: flex; align-items: center; gap: 10px; }
		.rule-id { font-family: 'JetBrains Mono', monospace; font-size: 13px; font-weight: 600; padding: 4px 10px; border-radius: 6px; }
		.rule-id.info { background: rgba(34,211,238,0.15); color: #22d3ee; }
		.rule-id.warning { background: rgba(251,191,36,0.15); color: #fbbf24; }
		.rule-id.high { background: rgba(248,113,113,0.15); color: #f87171; }
		.rule-id.critical { background: rgba(244,114,182,0.15); color: #f472b6; }
		.issue-message { font-size: 15px; color: #e4e4e7; flex: 1; }
		.issue-meta { display: flex; gap: 20px; font-size: 13px; color: #71717a; }
		.issue-meta .location { display: flex; align-items: center; gap: 6px; }
		.issue-details { margin-top: 12px; padding-top: 12px; border-top: 1px solid rgba(255,255,255,0.06); }
		.detail-row { display: flex; gap: 12px; font-size: 13px; margin-bottom: 6px; }
		.detail-label { color: #71717a; min-width: 100px; }
		.detail-value { color: #a1a1aa; }
		.detail-value.reason { color: #86efac; }
		.detail-value.recommendation { color: #67e8f9; }

		.no-issues {
			text-align: center; padding: 60px 40px;
			background: rgba(34,211,238,0.05);
			border: 1px dashed rgba(34,211,238,0.3); border-radius: 16px;
		}
		.no-issues-icon { font-size: 48px; margin-bottom: 16px; }
		.no-issues h3 { font-size: 20px; color: #22d3ee; margin-bottom: 8px; }
		.no-issues p { color: #71717a; }

		.footer { margin-top: 60px; padding-top: 24px; border-top: 1px solid rgba(255,255,255,0.06); text-align: center; color: #52525b; font-size: 13px; }
		.footer a { color: #8b5cf6; text-decoration: none; }

		@media (max-width: 768px) {
			.chart-section { grid-template-columns: 1fr; }
			.header { flex-direction: column; gap: 20px; text-align: center; }
			.meta { text-align: center; }
			.pie-chart { flex-direction: column; }
		}
	</style>
</head>
<body>
	<div class="container">
		<header class="header">
			<div class="logo">
				<div class="logo-icon">🔍</div>
				<div>
					<h1>gore-lint</h1>
					<span>Index Advisor Report</span>
				</div>
			</div>
			<div class="meta">
				<div><strong>Target:</strong> {{ .Report.Target }}</div>
				<div><strong>Generated:</strong> {{ .GeneratedAt }}</div>
			</div>
		</header>

		<div class="stats-grid">
			<div class="stat-card total">
				<div class="stat-label">Total Issues</div>
				<div class="stat-value">{{ .Report.Stats.Total }}</div>
			</div>
			<div class="stat-card info">
				<div class="stat-label">Info</div>
				<div class="stat-value">{{ .Report.Stats.Info }}</div>
			</div>
			<div class="stat-card warning">
				<div class="stat-label">Warnings</div>
				<div class="stat-value">{{ .Report.Stats.Warn }}</div>
			</div>
			<div class="stat-card high">
				<div class="stat-label">High</div>
				<div class="stat-value">{{ .Report.Stats.High }}</div>
			</div>
			<div class="stat-card critical">
				<div class="stat-label">Critical</div>
				<div class="stat-value">{{ .Report.Stats.Critical }}</div>
			</div>
		</div>

		{{ if gt .Report.Stats.Total 0 }}
		<div class="chart-section">
			<div class="chart-card">
				<div class="chart-title">Issue Distribution</div>
				<div class="pie-chart">
					{{ .ChartSVG }}
					<div class="legend">
						<div class="legend-item"><span class="legend-dot info"></span><span class="legend-label">Info</span><span class="legend-value">{{ .Report.Stats.Info }}</span></div>
						<div class="legend-item"><span class="legend-dot warning"></span><span class="legend-label">Warning</span><span class="legend-value">{{ .Report.Stats.Warn }}</span></div>
						<div class="legend-item"><span class="legend-dot high"></span><span class="legend-label">High</span><span class="legend-value">{{ .Report.Stats.High }}</span></div>
						<div class="legend-item"><span class="legend-dot critical"></span><span class="legend-label">Critical</span><span class="legend-value">{{ .Report.Stats.Critical }}</span></div>
					</div>
				</div>
			</div>
			<div class="chart-card">
				<div class="chart-title">Rules Triggered</div>
				<div class="rules-grid">
					{{ range $ruleID, $count := .RuleCounts }}
					<div class="rule-badge {{ ruleSeverityClass $ruleID }}">
						<span class="rule-id {{ ruleSeverityClass $ruleID }}">{{ $ruleID }}</span>
						<span style="color: #71717a;">×{{ $count }}</span>
					</div>
					{{ end }}
				</div>
			</div>
		</div>
		{{ end }}

		<div class="issues-section">
			<div class="section-header">
				<h2 class="section-title">Issues <span class="count">{{ .Report.Stats.Total }}</span></h2>
			</div>

			{{ if eq .Report.Stats.Total 0 }}
			<div class="no-issues">
				<div class="no-issues-icon">✅</div>
				<h3>No Issues Found</h3>
				<p>Your code follows best practices for database indexing!</p>
			</div>
			{{ else }}
				{{ range $sev, $issues := groupBySeverity .Report.Suggestions }}
				<div class="severity-group">
					<div class="severity-header">
						<span class="severity-indicator {{ $sev | sevClass }}"></span>
						<span class="severity-name {{ $sev | sevClass }}">{{ $sev | sevLabel }}</span>
						<span class="severity-count">{{ len $issues }}</span>
					</div>
					{{ range $issues }}
					<div class="issue-card">
						<div class="issue-header">
							<div class="issue-rule">
								<span class="rule-id {{ .Severity | sevClass }}">{{ .RuleID }}</span>
								<span class="issue-message">{{ .Message }}</span>
							</div>
						</div>
						<div class="issue-meta">
							{{ if .SourceFile }}
							<span class="location">📍 {{ .SourceFile }}:{{ .LineNumber }}</span>
							{{ end }}
						</div>
						{{ if or .Reason .Recommendation }}
						<div class="issue-details">
							{{ if .Reason }}
							<div class="detail-row">
								<span class="detail-label">Reason:</span>
								<span class="detail-value reason">{{ .Reason }}</span>
							</div>
							{{ end }}
							{{ if .Recommendation }}
							<div class="detail-row">
								<span class="detail-label">Recommendation:</span>
								<span class="detail-value recommendation">{{ .Recommendation }}</span>
							</div>
							{{ end }}
						</div>
						{{ end }}
					</div>
					{{ end }}
				</div>
				{{ end }}
			{{ end }}
		</div>

		<footer class="footer">
			<p>Generated by <a href="#">gore-lint</a> · Index Advisor for Go</p>
		</footer>
	</div>
</body>
</html>`

	funcMap := template.FuncMap{
		"sevClass":          severityClass,
		"sevLabel":          severityLabel,
		"groupBySeverity":    groupBySeverity,
		"ruleSeverityClass": func(ruleID string) string { return severityClass(ruleSeverity(ruleID)) },
	}

	t, err := template.New("report").Funcs(funcMap).Parse(tmpl)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

func generatePieChart(stats reportStats) string {
	if stats.Total == 0 {
		return `<svg width="160" height="160" viewBox="0 0 160 160"><circle cx="80" cy="80" r="60" fill="none" stroke="#333" stroke-width="20"/><text x="80" y="85" text-anchor="middle" fill="#71717a" font-size="14">No data</text></svg>`
	}

	type segment struct {
		value int
		color string
	}
	segments := []segment{
		{stats.Info, "#22d3ee"},
		{stats.Warn, "#fbbf24"},
		{stats.High, "#f87171"},
		{stats.Critical, "#f472b6"},
	}

	var paths []string
	cx, cy, r := 80.0, 80.0, 60.0
	startAngle := -90.0

	for _, seg := range segments {
		if seg.value == 0 {
			continue
		}
		percentage := float64(seg.value) / float64(stats.Total)
		angle := percentage * 360

		x1 := cx + r*math.Cos(degToRad(startAngle))
		y1 := cy + r*math.Sin(degToRad(startAngle))
		x2 := cx + r*math.Cos(degToRad(startAngle+angle))
		y2 := cy + r*math.Sin(degToRad(startAngle+angle))

		largeArc := 0
		if angle > 180.0 {
			largeArc = 1
		}

		path := fmt.Sprintf(`<path d="M %f %f A %f %f 0 %d 1 %f %f L %f %f Z" fill="%s"/>`, x1, y1, r, r, largeArc, x2, y2, cx, cy, seg.color)
		paths = append(paths, path)
		startAngle += angle
	}

	return fmt.Sprintf(`<svg width="160" height="160" viewBox="0 0 160 160">%s<circle cx="80" cy="80" r="35" fill="#1a1a2e"/></svg>`, strings.Join(paths, ""))
}

func degToRad(deg float64) float64 {
	return deg * math.Pi / 180
}
