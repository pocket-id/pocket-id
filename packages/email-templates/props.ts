import messages from '../../backend/resources/i18n/email/en.json';

export type MessageKey = keyof typeof messages;

// Returns a message, or with part only the text before (0), inside (1) or after (2) its {#link}…{/link}
type Translate = (key: MessageKey, vars?: Record<string, string>, part?: 0 | 1 | 2) => string;

export interface SharedProps {
	logoURL: string;
	appName: string;
	appURL: string;
	locale: string;
	t: Translate;
}

const linkTags = /\{#link\}|\{\/link\}/;

// Fills the English message with the sample values, so the preview server shows real text
const previewT: Translate = (key, vars = {}, part) => {
	const message = messages[key].replace(/\{(\w+)\}/g, (match, name: string) => vars[name] ?? match);
	return part === undefined ? message : message.split(linkTags)[part];
};

// Emits the Go template action that translates the message at send time, passing each value placeholder like {{.AppName}} as the argument (.AppName)
const templateT: Translate = (key, vars = {}, part) => {
	const call = part === undefined ? `.T "${key}"` : `.Part ${part} "${key}"`;
	const args = Object.entries(vars).map(
		([name, value]) => `"${name}" ${value.replace(/^\{\{(.*)\}\}$/, '($1)')}`
	);
	return `{{${[call, ...args].join(' ')}}}`;
};

export const sharedPreviewProps: SharedProps = {
	logoURL: 'https://pocket-id.org/img/logo.png',
	appName: 'Pocket ID',
	appURL: 'https://id.example.com',
	locale: 'en',
	t: previewT
};

export const sharedTemplateProps: SharedProps = {
	logoURL: '{{.LogoURL}}',
	appName: '{{.AppName}}',
	appURL: '{{.AppURL}}',
	locale: '{{.Locale}}',
	t: templateT
};
