import { render } from 'react-email';
import * as fs from 'node:fs';
import * as path from 'node:path';
import { defaultLocale, type Locale, locales } from './locales';

const outputDir = '../backend/resources/email-templates';

if (!fs.existsSync(outputDir)) {
	fs.mkdirSync(outputDir, { recursive: true });
}

function getTemplateName(filename: string): string {
	return filename.replace('.tsx', '');
}

// The English build keeps the original, unsuffixed file names so the backend's default
// templates (and existing tests/snapshots) are unaffected. Other locales get a "_<locale>"
// infix before the "_html"/"_text" suffix, e.g. "email-verification_fr_html.tmpl".
function localeInfix(locale: Locale): string {
	return locale === defaultLocale ? '' : `_${locale}`;
}

async function buildTemplateFile(
	Component: any,
	templateName: string,
	isPlainText: boolean,
	locale: Locale
) {
	const props = { ...Component.TemplateProps(locale), locale };
	const rendered = await render(Component(props), isPlainText ? { plainText: true } : {});

	// Normalize quotes
	const normalized = rendered.replace(/&quot;/g, '"');

	const goTemplate = `{{define "root"}}${normalized}{{end}}`;
	const suffix = isPlainText ? '_text.tmpl' : '_html.tmpl';
	const templatePath = path.join(outputDir, `${templateName}${localeInfix(locale)}${suffix}`);

	fs.writeFileSync(templatePath, goTemplate);
}

async function discoverAndBuildTemplates() {
	console.log('Discovering and building email templates...');

	const emailsDir = './emails';
	const files = fs.readdirSync(emailsDir);

	for (const file of files) {
		if (!file.endsWith('.tsx')) continue;

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

			for (const locale of locales) {
				await buildTemplateFile(Component, templateName, false, locale); // HTML
				await buildTemplateFile(Component, templateName, true, locale); // Text
			}

			console.log(`✓ Built ${templateName} (${locales.join(', ')})`);
		} catch (error) {
			console.error(`✗ Error building ${templateName}:`, error);
		}
	}
}

async function main() {
	await discoverAndBuildTemplates();
	console.log('All templates built successfully!');
}

main().catch(console.error);
