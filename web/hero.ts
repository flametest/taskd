import { heroui } from "@heroui/theme/plugin";

// HeroUI theme: brand color indigo #4F46E5 as primary, applied to both light
// and dark. Change DEFAULT here to rebrand.
export default heroui({
  themes: {
    light: {
      colors: {
        primary: { DEFAULT: "#4F46E5", foreground: "#FFFFFF" },
      },
    },
    dark: {
      colors: {
        primary: { DEFAULT: "#4F46E5", foreground: "#FFFFFF" },
      },
    },
  },
});
