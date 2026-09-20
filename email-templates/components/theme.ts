// Design tokens mirrored from the frontend's app.css (light theme) so emails match the app
// Email clients have no CSS variable or oklch support, so the values are the resolved hex equivalents
export const colors = {
  background: "#f5f5f5",
  card: "#ffffff",
  border: "#e5e5e5",
  muted: "#fafafa",
  foreground: "#0a0a0a",
  text: "#404040",
  mutedForeground: "#737373",
  primary: "#171717",
  primaryForeground: "#ffffff",
};

export const fonts = {
  sans: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif",
  serif: "Gloock, Georgia, 'Times New Roman', serif",
  mono: "'SF Mono', Menlo, Consolas, 'Liberation Mono', monospace",
};

export const radius = {
  card: "24px",
  box: "16px",
  pill: "9999px",
};
