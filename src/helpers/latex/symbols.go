// Package latex provides LaTeX-to-Unicode text conversion for math notation.
package latex

// commands maps LaTeX command sequences to their Unicode equivalents.
// Keys include the leading backslash (e.g. `\alpha`).
var commands = map[string]string{
	// Greek lowercase
	`\alpha`:   "α",
	`\beta`:    "β",
	`\gamma`:   "γ",
	`\delta`:   "δ",
	`\epsilon`: "ε",
	`\zeta`:    "ζ",
	`\eta`:     "η",
	`\theta`:   "θ",
	`\iota`:    "ι",
	`\kappa`:   "κ",
	`\lambda`:  "λ",
	`\mu`:      "μ",
	`\nu`:      "ν",
	`\xi`:      "ξ",
	`\pi`:      "π",
	`\rho`:     "ρ",
	`\sigma`:   "σ",
	`\tau`:     "τ",
	`\upsilon`: "υ",
	`\phi`:     "φ",
	`\chi`:     "χ",
	`\psi`:     "ψ",
	`\omega`:   "ω",

	// Greek uppercase
	`\Gamma`:  "Γ",
	`\Delta`:  "Δ",
	`\Theta`:  "Θ",
	`\Lambda`: "Λ",
	`\Xi`:     "Ξ",
	`\Pi`:     "Π",
	`\Sigma`:  "Σ",
	`\Phi`:    "Φ",
	`\Psi`:    "Ψ",
	`\Omega`:  "Ω",

	// Operators
	`\sum`:     "∑",
	`\prod`:    "∏",
	`\int`:     "∫",
	`\sqrt`:    "√",
	`\pm`:      "±",
	`\mp`:      "∓",
	`\times`:   "×",
	`\div`:     "÷",
	`\cdot`:    "·",
	`\neq`:     "≠",
	`\leq`:     "≤",
	`\geq`:     "≥",
	`\approx`:  "≈",
	`\infty`:   "∞",
	`\partial`: "∂",
	`\nabla`:   "∇",
	`\forall`:  "∀",
	`\exists`:  "∃",

	// Arrows
	`\rightarrow`:     "→",
	`\leftarrow`:      "←",
	`\Rightarrow`:     "⇒",
	`\Leftarrow`:      "⇐",
	`\leftrightarrow`: "↔",

	// Sets and logic
	`\in`:       "∈",
	`\notin`:    "∉",
	`\subset`:   "⊂",
	`\supset`:   "⊃",
	`\cup`:      "∪",
	`\cap`:      "∩",
	`\emptyset`: "∅",
	`\land`:     "∧",
	`\lor`:      "∨",
	`\neg`:      "¬",

	// Miscellaneous
	`\ldots`:  "…",
	`\cdots`:  "⋯",
	`\degree`: "°",
}

// superscripts maps single characters to their Unicode superscript equivalents.
var superscripts = map[rune]rune{
	'0': '⁰', '1': '¹', '2': '²', '3': '³', '4': '⁴',
	'5': '⁵', '6': '⁶', '7': '⁷', '8': '⁸', '9': '⁹',
	'n': 'ⁿ', 'i': 'ⁱ',
	'+': '⁺', '-': '⁻', '(': '⁽', ')': '⁾',
}

// subscripts maps single characters to their Unicode subscript equivalents.
var subscripts = map[rune]rune{
	'0': '₀', '1': '₁', '2': '₂', '3': '₃', '4': '₄',
	'5': '₅', '6': '₆', '7': '₇', '8': '₈', '9': '₉',
	'a': 'ₐ', 'e': 'ₑ', 'i': 'ᵢ', 'n': 'ₙ',
	'o': 'ₒ', 'r': 'ᵣ', 'u': 'ᵤ', 'v': 'ᵥ', 'x': 'ₓ',
	'+': '₊', '-': '₋', '=': '₌', '(': '₍', ')': '₎',
}

// sortedCommandKeys holds command keys ordered longest-first for greedy matching.
// Populated at init time via initSortedKeys.
var sortedCommandKeys []string

func init() {
	sortedCommandKeys = initSortedKeys(commands)
}
