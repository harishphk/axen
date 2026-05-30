// @ts-check
import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";

// https://astro.build/config
export default defineConfig({
  site: "https://axen.devhttps://axen.domains.workers.dev",
  integrations: [
    starlight({
      title: "Axen",
      social: [
        { icon: "github", label: "GitHub", href: "https://github.com/axen" },
      ],
      sidebar: [
        {
          label: "Getting Started",
          items: [
            { label: "Installation", slug: "getting-started/installation" },
            { label: "Quick Start", slug: "getting-started/quick-start" },
          ],
        },
        {
          label: "Guides",
          items: [
            { label: "Core Concepts", slug: "guides/core-concepts" },
            { label: "Configuration", slug: "guides/configuration" },
          ],
        },
        {
          label: "Reference",
          items: [
            { label: "Supported AI Tools", slug: "reference/supported-tools" },
          ],
        },
        {
          label: "CLI Reference",
          items: [{ autogenerate: { directory: "commands" } }],
        },
      ],
    }),
  ],
});
