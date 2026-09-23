import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    restoreMocks: true,
    projects: [
      {
        test: {
          name: "model",
          environment: "node",
          include: ["src/**/*.test.ts"],
          setupFiles: ["./vitest.setup.ts"],
          restoreMocks: true,
        },
      },
      {
        test: {
          name: "screens",
          environment: "jsdom",
          include: ["src/**/*.test.tsx"],
          setupFiles: ["./vitest.setup.ts", "./vitest.screens.ts"],
          restoreMocks: true,
        },
      },
    ],
  },
});
