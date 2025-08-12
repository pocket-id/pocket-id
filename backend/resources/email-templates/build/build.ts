import { render } from '@react-email/components';
import * as fs from 'node:fs';
import * as path from 'node:path';

const outputDir = '../';

if (!fs.existsSync(outputDir)) {
  fs.mkdirSync(outputDir, { recursive: true });
}

/**
 * Remove all `.tmpl` files from the configured output directory.
 *
 * Scans the module-level `outputDir` for files ending with `.tmpl` and deletes them synchronously.
 * Logs a summary of removed files and any errors encountered during the cleanup.
 */
function cleanupOldTemplates() {
  console.log('Cleaning up old template files...');
  
  try {
    const files = fs.readdirSync(outputDir);
    const templateFiles = files.filter(file => file.endsWith('.tmpl'));
    
    for (const file of templateFiles) {
      const filePath = path.join(outputDir, file);
      fs.unlinkSync(filePath);
      console.log(`✓ Removed ${file}`);
    }
    
    if (templateFiles.length === 0) {
      console.log('No old template files to clean up');
    } else {
      console.log(`Cleaned up ${templateFiles.length} old template files`);
    }
  } catch (error) {
    console.error('Error cleaning up old templates:', error);
  }
}

/**
 * Build sample props (placeholders) used to render email components for each template.
 *
 * Returns an object containing base placeholders (logoURL and appName) and, for known
 * templates, a `data` object with template-specific placeholder fields.
 *
 * Recognized `templateName` values and their `data` fields:
 * - "new-signin": { city, country, ipAddress, device, dateTime }
 * - "one-time-access": { code, loginLink, buttonCodeLink, expirationString }
 * - "api-key-expiring-soon": { name, apiKeyName, expiresAt }
 * - "test": no additional data
 *
 * If `templateName` is not recognized the function returns only the base placeholders.
 *
 * @param templateName - The template identifier (derived from the filename, e.g. "new-signin").
 * @returns An object suitable for passing to the email component when rendering sample output.
 */
function getSampleProps(templateName: string) {
  const baseProps = {
    logoURL: "LOGOURL_PLACEHOLDER",
    appName: "APPNAME_PLACEHOLDER",
  };

  const dataMap: Record<string, any> = {
    'new-signin': {
      city: "CITY_PLACEHOLDER",
      country: "COUNTRY_PLACEHOLDER",
      ipAddress: "IPADDRESS_PLACEHOLDER",
      device: "DEVICE_PLACEHOLDER",
      dateTime: "DATETIME_PLACEHOLDER",
    },
    'one-time-access': {
      code: "CODE_PLACEHOLDER",
      loginLink: "LOGINLINK_PLACEHOLDER",
      buttonCodeLink: "BUTTONCODELINK_PLACEHOLDER",
      expirationString: "EXPIRATIONSTRING_PLACEHOLDER",
    },
    'api-key-expiring-soon': {
      name: "NAME_PLACEHOLDER",
      apiKeyName: "APIKEYNAME_PLACEHOLDER",
      expiresAt: "EXPIRESAT_PLACEHOLDER",
    },
    'test': {},
  };

  const data = dataMap[templateName] || {};
  
  return Object.keys(data).length > 0 
    ? { ...baseProps, data } 
    : baseProps;
}

/**
 * Build the list of placeholder replacements to convert rendered HTML into Go template syntax.
 *
 * Uses a base set of replacements (logo and app name) plus template-specific mappings for
 * common placeholders (e.g., CITY_PLACEHOLDER → {{.Data.City}}). For some templates the list
 * includes regex patterns that match multiple variants (for example `BUTTONCODELINK_[A-Za-z0-9]+`
 * is mapped to the single `{{.Data.LoginLinkWithCode}}` replacement).
 *
 * @param templateName - The canonical template identifier (e.g., `"new-signin"`, `"one-time-access"`) used to select template-specific replacements.
 * @returns An array of replacement descriptors; each item has `search` (a RegExp to find placeholders in the HTML) and `replace` (the Go template string to substitute).
 */
function getReplacements(templateName: string) {
  const baseReplacements = [
    { search: /LOGOURL_PLACEHOLDER/g, replace: '{{.LogoURL}}' },
    { search: /APPNAME_PLACEHOLDER/g, replace: '{{.AppName}}' },
  ];

  const dataReplacements: Record<string, Array<{ search: RegExp; replace: string }>> = {
    'new-signin': [
      { search: /CITY_PLACEHOLDER/g, replace: '{{.Data.City}}' },
      { search: /COUNTRY_PLACEHOLDER/g, replace: '{{.Data.Country}}' },
      { search: /IPADDRESS_PLACEHOLDER/g, replace: '{{.Data.IPAddress}}' },
      { search: /DEVICE_PLACEHOLDER/g, replace: '{{.Data.Device}}' },
      { search: /DATETIME_PLACEHOLDER/g, replace: '{{.Data.DateTime.Format "January 2, 2006 at 3:04 PM MST"}}' },
    ],
    'one-time-access': [
      { search: /CODE_PLACEHOLDER/g, replace: '{{.Data.Code}}' },
      { search: /LOGINLINK_PLACEHOLDER/g, replace: '{{.Data.LoginLink}}' },
      { search: /BUTTONCODELINK_PLACEHOLDER/g, replace: '{{.Data.LoginLinkWithCode}}' },
      { search: /EXPIRATIONSTRING_PLACEHOLDER/g, replace: '{{.Data.ExpirationString}}' },
      { search: /BUTTONCODELINK_[A-Za-z0-9]+/g, replace: '{{.Data.LoginLinkWithCode}}' },
    ],
    'api-key-expiring-soon': [ 
      { search: /NAME_PLACEHOLDER/g, replace: '{{.Data.Name}}' },
      { search: /APIKEYNAME_PLACEHOLDER/g, replace: '{{.Data.ApiKeyName}}' },
      { search: /EXPIRESAT_PLACEHOLDER/g, replace: '{{.Data.ExpiresAt.Format "2006-01-02 15:04:05 MST"}}' },
    ],
    'test': [],
  };

  return [...baseReplacements, ...(dataReplacements[templateName] || [])];
}

/**
 * Returns a plain-text Go template for the given email template name.
 *
 * Looks up a pre-defined text template keyed by `templateName` (examples: `new-signin`, `one-time-access`, `api-key-expiring-soon`, `test`) and returns it as a string containing a Go `{{define "root"}}...{{end}}` block. If no specific template exists for `templateName`, a generic fallback template is returned where the title is derived from the name (hyphen-separated words are capitalized and joined).
 *
 * @param templateName - The canonical template identifier (derived from the TSX filename, e.g. `new-signin`)
 * @returns A text email template formatted as a Go template string.
 */
function getTextTemplate(templateName: string) {
  const title = templateName
    .split('-')
    .map(word => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');

  const templates: Record<string, string> = {
    'new-signin': `{{ define "root" -}}
New Sign-In Detected
====================

{{ if and .Data.City .Data.Country }}
Approximate Location: {{ .Data.City }}, {{ .Data.Country }}
{{ end }}
IP Address: {{ .Data.IPAddress }}
Device:     {{ .Data.Device }}
Time:       {{ .Data.DateTime.Format "2006-01-02 15:04:05 UTC"}}

This sign-in was detected from a new device or location. If you recognize
this activity, you can safely ignore this message. If not, please review
your account and security settings.

--
This is automatically sent email from {{.AppName}}.
{{ end -}}`,

    'one-time-access': `{{ define "root" -}}
Login Code
====================

Click the link below to sign in to {{ .AppName }} with a login code. This code expires in {{.Data.ExpirationString}}.

{{ .Data.LoginLinkWithCode }}

Or visit {{ .Data.LoginLink }} and enter the the code "{{ .Data.Code }}".

--
This is automatically sent email from {{.AppName}}.
{{ end -}}`,

    'api-key-expiring-soon': `{{ define "root" -}}
API Key Expiring Soon
====================

Hello {{ .Data.Name }},

This is a reminder that your API key "{{ .Data.ApiKeyName }}" will expire on {{ .Data.ExpiresAt.Format "2006-01-02 15:04:05 MST" }}.

Please generate a new API key if you need continued access.

--
This is automatically sent email from {{.AppName}}.
{{ end -}}`,

    'test': `{{ define "root" -}}
This is a test email.

--
This is automatically sent email from {{.AppName}}.
{{ end -}}`,
  };

  return templates[templateName] || `{{ define "root" -}}
${title}
${'='.repeat(title.length)}

Email content goes here.

--
This is automatically sent email from {{.AppName}}.
{{ end -}}`;
}

/**
 * Injects template-specific Go conditionals into rendered HTML.
 *
 * For the "new-signin" template, wraps the "Approximate Location" HTML block with
 * a Go `{{if and .Data.City .Data.Country}}...{{end}}` conditional so the block
 * is only included when both City and Country are present. For other templates
 * or when no matching block is found, returns the input HTML unchanged.
 *
 * @param templateName - Name of the template (e.g., "new-signin") that may require conditionals
 * @param html - Rendered HTML string to be modified
 * @returns The potentially modified HTML string with Go conditionals applied where appropriate
 */
function applyConditionals(templateName: string, html: string): string {
  if (templateName === 'new-signin') {
    // Wrap location data in conditional
    const locationRegex = /(<div[^>]*>[\s\S]*?<.*?>Approximate Location<[\s\S]*?{{\.Data\.City}}, {{\.Data\.Country}}[\s\S]*?<\/div>)/;
    const match = html.match(locationRegex);
    if (match) {
      return html.replace(locationRegex, `{{if and .Data.City .Data.Country}}${match[1]}{{end}}`);
    }
  }
  return html;
}

/**
 * Convert a TSX email filename into the canonical template name used by the build.
 *
 * Removes the `.tsx` extension and an optional `-email` suffix from `filename`.
 *
 * @param filename - TSX filename from the emails directory (e.g., `new-signin-email.tsx`)
 * @returns The template name (e.g., `new-signin`)
 */
function getTemplateName(filename: string): string {
  return filename
    .replace('.tsx', '')
    .replace('-email', '');
}

/**
 * Discover React Email components in ./emails, render them with sample props, convert the output to Go template syntax, and write HTML and text .tmpl files to the output directory.
 *
 * Scans the local ./emails directory for .tsx files (skipping files containing "base-template" or "components"), dynamically imports each component, renders it with template-specific sample props, applies placeholder replacements and special conditionals, maps the template name for the Go backend (e.g., "new-signin" -> "login-with-new-device"), and writes two files per template: `<name>_html.tmpl` (wrapped in a `{{define "root"}}...{{end}}` block) and `<name>_text.tmpl`.
 *
 * Side effects:
 * - Reads the ./emails directory.
 * - Performs dynamic imports of modules within that directory.
 * - Writes template files into the configured output directory.
 * - Logs progress and errors to the console.
 *
 * @returns A Promise that resolves when discovery and file generation complete.
 */
async function discoverAndBuildTemplates() {
  console.log('Discovering and building email templates...');
  
  const emailsDir = './emails';
  const files = fs.readdirSync(emailsDir);
  
  for (const file of files) {
    // Skip base template and components
    if (file.endsWith('.tsx') && 
        !file.includes('base-template') && 
        !file.includes('components')) {
      
      const templateName = getTemplateName(file);
      const modulePath = `./${emailsDir}/${file}`; // Fix: use relative path with ./
      
      console.log(`Building ${templateName}...`);
      
      try {
        // Dynamic import with proper relative path
        const module = await import(modulePath);
        const Component = module.default || module[Object.keys(module)[0]];
        
        if (!Component) {
          console.error(`✗ No component found in ${file}`);
          continue;
        }
        
        // Get sample props and render
        const sampleProps = getSampleProps(templateName);
        const html = await render(Component(sampleProps));
        
        // Apply replacements
        const replacements = getReplacements(templateName);
        let goTemplateHtml = html;
        
        for (const replacement of replacements) {
          goTemplateHtml = goTemplateHtml.replace(replacement.search, replacement.replace);
        }
        
        // Apply conditionals
        goTemplateHtml = applyConditionals(templateName, goTemplateHtml);
        
        // Convert template name for Go backend
        const goTemplateName = templateName === 'new-signin' ? 'login-with-new-device' : templateName;
        
        // Write HTML template
        const goHtmlTemplate = `{{define "root"}}${goTemplateHtml}{{end}}`;
        const htmlTemplatePath = path.join(outputDir, `${goTemplateName}_html.tmpl`);
        fs.writeFileSync(htmlTemplatePath, goHtmlTemplate);
        
        // Write text template
        const textTemplate = getTextTemplate(templateName);
        const textTemplatePath = path.join(outputDir, `${goTemplateName}_text.tmpl`);
        fs.writeFileSync(textTemplatePath, textTemplate);
        
        console.log(`✓ Generated ${htmlTemplatePath} and ${textTemplatePath}`);
      } catch (error) {
        console.error(`✗ Error building ${templateName}:`, error);
      }
    }
  }
}

/**
 * Orchestrates the template build process: removes old templates, builds new ones, and logs completion.
 *
 * This async entrypoint first runs cleanupOldTemplates() to delete existing .tmpl files, then awaits
 * discoverAndBuildTemplates() to discover, render, convert, and write new Go-compatible templates.
 *
 * @returns A promise that resolves when the build process finishes.
 */
async function main() {
  cleanupOldTemplates();
  await discoverAndBuildTemplates();
  console.log('All templates built successfully!');
}

main().catch(console.error);