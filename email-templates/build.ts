import { render } from "@react-email/components";
import * as fs from "node:fs";
import * as path from "node:path";

const outputDir = "../backend/resources/email-templates";

if (!fs.existsSync(outputDir)) {
  fs.mkdirSync(outputDir, { recursive: true });
}

function getTemplateName(filename: string): string {
  return filename.replace(".tsx", "");
}

async function buildTemplateFile(
  Component: any,
  templateName: string,
  isPlainText: boolean
) {
  const rendered = await render(Component(Component.TemplateProps), {
    plainText: isPlainText,
  });

  // Normalize quotes
  const normalized = rendered.replace(/&quot;/g, '"');

  const goTemplate = `{{define "root"}}${normalized}{{end}}`;
  const suffix = isPlainText ? "_text.tmpl" : "_html.tmpl";
  const templatePath = path.join(outputDir, `${templateName}${suffix}`);

  fs.writeFileSync(templatePath, goTemplate);
}

async function discoverAndBuildTemplates() {
  console.log("Discovering and building email templates...");

  const emailsDir = "./emails";
  const files = fs.readdirSync(emailsDir);

  for (const file of files) {
    if (!file.endsWith(".tsx")) continue;

    const templateName = getTemplateName(file);
    const modulePath = `./${emailsDir}/${file}`;

    console.log(`Building ${templateName}...`);

    try {
      const module = await import(modulePath);
      const Component = module.default || module[Object.keys(module)[0]];

      if (!Component) {
        console.error(`✗ No component found in ${file}`);
        continue;
      }

      if (!Component.TemplateProps) {
        console.error(`✗ No TemplateProps found in ${file}`);
        continue;
      }

      await buildTemplateFile(Component, templateName, false); // HTML
      await buildTemplateFile(Component, templateName, true); // Text

      console.log(`✓ Built ${templateName}`);
    } catch (error) {
      console.error(`✗ Error building ${templateName}:`, error);
    }
  }
}

async function main() {
  await discoverAndBuildTemplates();
  console.log("All templates built successfully!");
}

main().catch(console.error);
