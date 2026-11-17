import base from '@playwright/test';
import { createServer } from 'node:net';

export type CapturedEmail = {
	recipients: string[];
	subject: string;
	body: string;
};

export type SmtpSink = {
	emails: CapturedEmail[];
	close: () => Promise<void>;
};

const smtpSinkPort = 18125;

// The application configuration that makes Pocket ID send its emails to the sink
export const smtpSinkConfig = {
	smtpHost: 'host.docker.internal',
	smtpPort: String(smtpSinkPort),
	smtpFrom: 'pocket-id@test.com',
	smtpTls: 'none'
};

// A test that gets an SMTP sink, which is closed again after the test
export const test = base.extend<{ smtp: SmtpSink }>({
	smtp: async ({}, use) => {
		const sink = await startSmtpSink(smtpSinkPort);
		try {
			await use(sink);
		} finally {
			await sink.close();
		}
	}
});

// Starts a minimal SMTP server that accepts every message, so tests can read the emails Pocket ID sends
// Docker reaches it through the host gateway, like the back-channel logout receiver
async function startSmtpSink(port: number): Promise<SmtpSink> {
	const emails: CapturedEmail[] = [];

	const server = createServer((socket) => {
		// Latin-1 keeps one character per byte, so the UTF-8 text can be decoded once the message is complete
		let buffer = '';
		let inData = false;
		let recipients: string[] = [];

		socket.write('220 smtp-sink ESMTP\r\n');
		socket.on('data', (chunk) => {
			buffer += chunk.toString('latin1');

			while (true) {
				// The message ends with a line that only holds a dot
				if (inData) {
					const end = buffer.indexOf('\r\n.\r\n');
					if (end === -1) return;

					emails.push(parseMessage(recipients, buffer.slice(0, end)));
					buffer = buffer.slice(end + 5);
					inData = false;
					recipients = [];
					socket.write('250 OK\r\n');
					continue;
				}

				const lineEnd = buffer.indexOf('\r\n');
				if (lineEnd === -1) return;
				const line = buffer.slice(0, lineEnd);
				buffer = buffer.slice(lineEnd + 2);

				// Answer just enough of the protocol for a client that neither authenticates nor uses TLS
				const command = line.slice(0, 4).toUpperCase();
				if (command === 'EHLO' || command === 'HELO') {
					socket.write('250 smtp-sink\r\n');
				} else if (command === 'RCPT') {
					recipients.push(line.match(/<(.*)>/)?.[1] ?? '');
					socket.write('250 OK\r\n');
				} else if (command === 'DATA') {
					inData = true;
					socket.write('354 End data with <CR><LF>.<CR><LF>\r\n');
				} else if (command === 'QUIT') {
					socket.end('221 Bye\r\n');
					return;
				} else {
					socket.write('250 OK\r\n');
				}
			}
		});
	});

	await new Promise<void>((resolve, reject) => {
		server.once('error', reject);
		server.listen(port, '0.0.0.0', resolve);
	});

	return {
		emails,
		close: () =>
			new Promise<void>((resolve, reject) => server.close((e) => (e ? reject(e) : resolve())))
	};
}

// Returns the newest email sent to the recipient
export function lastEmailTo(sink: SmtpSink, recipient: string) {
	const email = sink.emails.findLast((e) => e.recipients.includes(recipient));
	if (!email) throw new Error(`No email was sent to ${recipient}`);
	return email;
}

// Returns the path of the newest email verification link sent to the recipient
export function verificationLinkPath(sink: SmtpSink, recipient: string) {
	const token = lastEmailTo(sink, recipient).body.match(
		/\/verify-email\?token=([A-Za-z0-9]+)/
	)?.[1];
	if (!token) throw new Error(`No verification link was sent to ${recipient}`);
	return `/verify-email?token=${token}`;
}

function parseMessage(recipients: string[], raw: string): CapturedEmail {
	// Unfold the headers and join the encoded words a long subject is split into
	const headers = raw.slice(0, raw.indexOf('\r\n\r\n')).replace(/\r\n[ \t]/g, ' ');
	const subject = (headers.match(/^Subject: (.*)$/m)?.[1] ?? '')
		.replace(/\?=\s+=\?/g, '?==?')
		.replace(/=\?utf-8\?q\?(.*?)\?=/gi, (_, word: string) =>
			decodeQuotedPrintable(word.replaceAll('_', ' '))
		);

	return { recipients, subject: utf8(subject), body: utf8(decodeQuotedPrintable(raw)) };
}

function decodeQuotedPrintable(value: string) {
	return value
		.replace(/=\r\n/g, '')
		.replace(/=([0-9A-F]{2})/g, (_, hex: string) => String.fromCharCode(parseInt(hex, 16)));
}

function utf8(latin1: string) {
	return Buffer.from(latin1, 'latin1').toString('utf8');
}
