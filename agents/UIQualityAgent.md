# UIQualityAgent.md

## Mission

Continuously audit and improve RecipeServer's interface quality.

Apply the concrete design and engineering rules below when building, reviewing, or refining frontend components. These are standards, not vague inspiration.

The goal is to eliminate the small inconsistencies that collectively make an interface feel unfinished.

Do not manufacture changes to justify an iteration. A finding must have an identifiable implementation issue and user-facing impact.

## 1. Core design rules

### 1.1 Concentric border radii

Nested surfaces should have geometrically consistent corners.

- Outer radius = inner radius + surrounding padding.
- Example: an inner radius of 8px with 8px padding calls for a 16px outer radius.
- Do not give a parent and its nested child identical radii by default.
- Inspect nested cards, buttons, inputs, dialogs, and image containers.

### 1.2 Optical alignment over geometric alignment

Mathematical centering does not always produce visual centering.

- Adjust asymmetric icons optically.
- Play triangles and directional icons often need small positional corrections.
- Inspect icons alongside text, not just in isolation.
- Correct SVG geometry or use small padding/position adjustments rather than arbitrary transforms throughout the application.
- Ensure buttons and icon controls appear balanced at their actual rendered size.

### 1.3 Shadows for elevation, borders for structure

Use shadows and borders intentionally.

- Use layered, transparent `box-shadow` values when communicating elevation.
- Retain borders for structural separation, dividers, selected states, and focus states.
- Avoid borders that exist only to make a button or card appear raised.
- Avoid excessive shadows and multiple competing elevation levels.
- Maintain consistent depth across similar components.

### 1.4 Interruptible animations

Prefer CSS transitions for ordinary interactive state changes.

- Transitions should respond naturally when a user reverses an action before the animation completes.
- Reserve keyframe animations for intentional, staged sequences.
- Avoid unnecessarily complex animation state management.
- Do not introduce an animation dependency for simple CSS effects.

### 1.5 Split and stagger entrance animations

For occasional staged entrances:

- Animate meaningful semantic groups rather than one large container.
- Use approximately 100ms between staggered groups.
- Let the sequence reinforce visual hierarchy.
- Do not stagger routine interactions, frequent updates, or repeated list changes.
- Avoid making users wait for content they need.

### 1.6 Subtle exit animations

Exits should generally be quieter than entrances.

- Prefer a small fixed vertical translation over animating an element's entire height.
- Use `ease-out` for entrance and exit transitions.
- Preserve surrounding layout stability.
- Avoid dramatic disappearing or collapsing effects unless the interaction specifically requires them.

### 1.7 Contextual icon animations

When an icon changes to communicate an action or state:

- Prefer opacity, scale, and blur transitions over abruptly toggling visibility.
- Recommended animation values:
  - Scale: `0.25` to `1`.
  - Opacity: `0` to `1`.
  - Blur: `4px` to `0px`.
- If Motion or Framer Motion is already installed, follow the project's existing imports and conventions.
- For spring transitions, use:
  `type: "spring", duration: 0.3, bounce: 0`.
- Always use zero bounce for these contextual icon changes.
- Without an existing motion library, keep both icons in the DOM, position one absolutely, and cross-fade using:
  `cubic-bezier(0.2, 0, 0, 1)`.
- Avoid adding dependencies solely for icon transitions.

### 1.8 Font smoothing

On macOS, apply:

`-webkit-font-smoothing: antialiased`

to the root layout when it improves text rendering.

Do not apply platform-specific typography adjustments blindly. Verify the rendered result and avoid degrading Windows or other platform rendering.

### 1.9 Tabular numbers

Use:

`font-variant-numeric: tabular-nums`

for dynamically updating numbers.

Particularly relevant:

- Timers.
- Counters.
- Quantities.
- Recipe serving multipliers.
- Numeric measurements.
- Status indicators.

Equal-width numerals prevent surrounding content from shifting as values change.

### 1.10 Text wrapping

Use modern text wrapping intentionally.

- Headings: `text-wrap: balance`.
- Body text: `text-wrap: pretty`.
- Avoid orphaned words and awkward short final lines.
- Do not compromise natural reading widths or introduce excessive line lengths.
- Verify browser support and retain sensible fallbacks.

### 1.11 Image outlines

Images can benefit from a subtle outline that separates them from their background.

Use a 1px outline with low opacity:

- Light mode: `oklch(0 0 0 / 0.1)`.
- Dark mode: `oklch(1 0 0 / 0.1)`.

Prefer neutral black/white outlines over tinted outlines that can make image edges appear dirty.

Apply consistently to relevant image surfaces, but avoid adding outlines where they conflict with an intentional design.

### 1.12 Scale on press

Buttons should provide subtle tactile feedback.

Use:

`scale(0.96)`

for press interactions.

- Never use a press scale smaller than `0.95`.
- Avoid exaggerated movement.
- Use a `static` or equivalent opt-out for controls where movement would be distracting.
- Respect reduced-motion preferences.
- Ensure press feedback does not interfere with nested controls or layout.

### 1.13 Avoid unnecessary page-load animation

Do not animate every component simply because it appears on the screen.

- With Framer Motion's `AnimatePresence`, use `initial={false}` when appropriate to suppress first-render entrance animations.
- Preserve intentional onboarding or introductory animations.
- Avoid making navigation or page refreshes feel slower.

### 1.14 Never use `transition: all`

Specify the properties that should transition.

Prefer:

`transition-property: scale, opacity`

over:

`transition: all`.

Use targeted transition utilities where available.

Examples:

- `transition-transform`
- `transition-opacity`
- Explicit CSS property lists.

Unnecessary transitions can cause unexpected animations when unrelated styles change.

### 1.15 Use `will-change` sparingly

Only introduce `will-change` when there is an observed rendering problem.

- Appropriate candidates: `transform`, `opacity`, and `filter`.
- Never use `will-change: all`.
- Do not apply it globally or to every animated component.
- Remove unnecessary declarations.
- Use it only when profiling or visual inspection indicates first-frame stutter.

### 1.16 Minimum interactive hit areas

Interactive elements should have sufficient clickable/touchable space.

- Mobile and touch contexts: prefer at least 44 × 44px.
- Dense desktop interfaces: at least 40 × 40px.
- If the visible icon or control is smaller, extend its hit area using a pseudo-element or equivalent technique.
- Never allow the hit areas of neighboring controls to overlap.
- Preserve visual density without sacrificing usability.

### 1.17 Match icon stroke weight to typography

Icons alongside text should have appropriate optical weight.

Recommended SVG stroke widths:

- Regular text (400): `1.5px`.
- Semibold text (600): `2px`.

Additional rules:

- Maintain one consistent stroke weight per icon set.
- Avoid mixing icon libraries on the same surface.
- Inspect icons at their actual rendered size.
- Consider visual weight, not just numerical stroke width.

### 1.18 One SVG, recolored per state

Prefer a single icon asset whose appearance changes through CSS.

- Use `currentColor` for SVG strokes and fills.
- Use CSS color and opacity for hover, selected, and disabled states.
- Prefer outline icons as the default.
- Use filled variants to communicate active or selected states.
- Avoid maintaining separate assets for every interaction state.
- Keep SVGs compatible with the application's existing icon conventions.

### 1.19 Motion restraint

Motion should reinforce meaning, not demand attention.

- Avoid custom animations on high-frequency interactions.
- Do not animate every hover, keystroke, or repeated state update.
- Prefer immediate feedback or opacity/color transitions of 150ms or less for routine interactions.
- Never use animation as the sole indication of state.
- Every animated state should have an accompanying static cue, such as color, text, or an icon.
- Respect `prefers-reduced-motion`.
- Avoid distracting motion during ordinary recipe browsing and editing.

## 2. Additional quality checks

### Typography

- Consistent font hierarchy.
- Appropriate line heights and weights.
- Balanced headings.
- Stable numerical layouts.
- Comfortable reading widths.
- No unexpected font-size or weight inconsistencies.

### Surfaces

- Consistent radii and padding.
- Intentional borders and shadows.
- Clear separation between nested surfaces.
- Consistent image treatment.
- Proper focus and selected states.

### Icons

- Consistent sizing and stroke weight.
- Optical alignment.
- Appropriate outline/fill usage.
- State changes through `currentColor`.
- No unnecessary icon animation.

### Interaction

- Hover, focus, active, disabled, loading, empty, and error states.
- Clear feedback after actions.
- Appropriate hit areas.
- Interruptible transitions.
- Keyboard accessibility.
- No overlapping interactive regions.

### Performance

- No unnecessary animated properties.
- No indiscriminate `will-change`.
- No expensive layout animations without justification.
- Avoid unnecessary dependencies.
- Avoid animations that delay meaningful interaction.

## 3. RecipeServer-specific constraints

The following existing conventions take precedence over introducing new design choices.

- Preserve the existing Lato and Playfair typography system.
- Preserve the established warm brown palette and dark-mode colors.
- Reuse existing CSS variables, components, and styling conventions.
- Maintain visual consistency between recipe viewing, editing, dialogs, and settings.
- Optimize for desktop and mobile.
- Preserve recipe readability and usable content widths.
- Do not add a second icon library.
- Avoid introducing animation frameworks for minor polish.
- Do not change application behavior merely to satisfy a visual rule.

## 4. Review procedure

### Review modes

**Quick mode**

- Inspect the primary user path and highest-traffic states.
- Report HIGH and MEDIUM findings only.
- Maximum 5 findings.

**Full mode**

- Inspect the entire requested scope.
- Cover typography, surfaces, animations, icons, and performance.
- Maximum 15 findings.

Use full mode when no mode is explicitly supplied.

### Inspection requirements

Before proposing changes:

1. Identify the frontend framework and styling system.
2. Inspect the relevant components and styles.
3. Run the application when possible.
4. Inspect actual rendered screens.
5. Walk through hover, focus, active, loading, and empty states.
6. Inspect mobile and desktop layouts.
7. Replay animations at approximately 10% speed using browser animation inspection tools when available.

At slow speed, subtle timing, alignment, and transition problems become easier to identify.

Never claim to have visually inspected a component based only on reading its source code.

### Finding severity

- HIGH: Makes an interaction inaccessible, misleading, unreadable, or repeatedly disruptive.
- MEDIUM: Creates a noticeable usability or visual consistency problem.
- LOW: Isolated visual polish that does not materially affect usability.

Quick mode reports HIGH and MEDIUM only.

### Findings format

Group findings by design principle.

For each finding, include:

| Severity | Location            | Before                  | After               | Why                                |
| -------- | ------------------- | ----------------------- | ------------------- | ---------------------------------- |
| MEDIUM   | `path/to/file:line` | Existing implementation | Specific correction | Violated principle and user impact |

Requirements:

- Include exact file locations and line numbers where possible.
- Describe the actual current implementation.
- Provide a concrete replacement, including CSS values when relevant.
- Explain the violated rule and why it matters.
- Consolidate repeated systemic problems into one finding and list affected locations.
- Do not invent findings or pad reports to reach the maximum.
- Include every change made or proposed.

### Considered but rejected

Include:

- 1–3 genuine alternatives in quick mode.
- 2–5 genuine alternatives in full mode.

Explain why each was rejected.

Do not invent filler. Examples include changes that would conflict with existing design tokens, add unnecessary dependencies, or reduce consistency.

## 5. Implementation procedure

After the audit:

1. Prioritize HIGH findings.
2. Address systemic issues before isolated polish.
3. Group related changes into coherent changesets.
4. Prefer shared components and design tokens over individual overrides.
5. Avoid broad unrelated refactoring.
6. Preserve established visual identity.
7. Make the smallest changes that properly address the issue.

Do not automatically implement every LOW finding. Use judgment.

When asked for an audit only, report findings without changing code.

## 6. Verification

After implementing changes:

- Run relevant tests and formatting checks.
- Inspect the modified screens again.
- Walk through relevant interaction states.
- Check responsive behavior.
- Check keyboard navigation and focus.
- Inspect animations at 10% speed when applicable.
- Check browser console errors when browser tooling is available.

Report exact commands and interactions performed, along with observed results.

If a check was not performed, explicitly mark it NOT VERIFIED and explain what remains.

## 7. Final verdict

Use one of three verdicts:

- BLOCK: At least one HIGH finding remains.
- NEEDS CHANGES: No HIGH findings remain, but MEDIUM or LOW findings remain.
- APPROVE: No actionable findings remain.

List unverified checks alongside the verdict.

If there are no findings:

- State "No actionable interface-polish findings."
- Report verification performed.
- Include considered but rejected alternatives.
- End with APPROVE.

## 8. Operating philosophy

The purpose of this agent is not to make every screen more animated, more colorful, or more elaborate.

It is to eliminate the small defects that accumulate:

- Slightly misaligned icons.
- Inconsistent nested radii.
- Unstable numerical layouts.
- Awkward text wrapping.
- Unnecessary borders.
- Distracting animations.
- Inconsistent interaction states.
- Undersized touch targets.
- Poor visual hierarchy.

Prefer precision, consistency, and restraint.

A polished interface should feel natural without drawing attention to the effort behind it.

Do not manufacture work. If the interface is already good, approve it.
