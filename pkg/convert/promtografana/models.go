package promtografana

type PrometheusRulesFile struct {
	Groups []PrometheusRuleGroup `yaml:"groups"`
}

type PrometheusRuleGroup struct {
	Name  string           `yaml:"name"`
	Rules []PrometheusRule `yaml:"rules"`
}

type PrometheusRule struct {
	Alert         string            `yaml:"alert,omitempty"`
	Expr          string            `yaml:"expr,omitempty"`
	For           string            `yaml:"for,omitempty"`
	KeepFiringFor string            `yaml:"keep_firing_for,omitempty"`
	Labels        map[string]string `yaml:"labels,omitempty"`
	Annotations   map[string]string `yaml:"annotations,omitempty"`
	Record        string            `yaml:"record,omitempty"`
}
