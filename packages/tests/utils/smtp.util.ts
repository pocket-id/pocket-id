import { createServer } from 'node:net';

export type CapturedEmail = {
	recipients: string[];
	body: string;
};

export type SmtpSink = {
	emails: CapturedEmail[];
	close: () => Promise<void>;
};

// Starts a minimal SMTP server that accepts every message, so tests can read the emails Pocket ID sends
// Docker reaches it through the host gateway, like the back-channel logout receiver
export async function startSmtpSink(port: number): Promise<SmtpSink> {
	const emails: CapturedEmail[] = [];

	const server = createServer((socket) => {
		let buffer = '';
		let inData = false;
		let recipients: string[] = [];

		socket.write('220 smtp-sink ESMTP\r\n');
		socket.on('data', (chunk) => {
			buffer += chunk.toString();

			while (true) {
				// The message ends with a line that only holds a dot
				if (inData) {
					const end = buffer.indexOf('\r\n.\r\n');
					if (end === -1) return;

					emails.push({ recipients, body: decodeQuotedPrintable(buffer.slice(0, end)) });
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

// Returns the path of the newest email verification link sent to the recipient
export function verificationLinkPath(sink: SmtpSink, recipient: string) {
	const email = sink.emails.findLast((e) => e.recipients.includes(recipient));
	const token = email?.body.match(/\/verify-email\?token=([A-Za-z0-9]+)/)?.[1];
	if (!token) throw new Error(`No verification link was sent to ${recipient}`);
	return `/verify-email?token=${token}`;
}

function decodeQuotedPrintable(value: string) {
	return value
		.replace(/=\r\n/g, '')
		.replace(/=([0-9A-F]{2})/g, (_, hex: string) => String.fromCharCode(parseInt(hex, 16)));
}
