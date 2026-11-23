import { Link, Text } from 'react-email';
import { colors } from './theme';

interface TextProps {
	children: React.ReactNode;
	style?: React.CSSProperties;
}

export const Muted = ({ children, style }: TextProps) => (
	<Text style={{ ...mutedStyle, ...style }}>{children}</Text>
);

export const MutedLink = ({ href, children }: { href: string; children: React.ReactNode }) => (
	<Link href={href} style={mutedLinkStyle}>
		{children}
	</Link>
);

const mutedStyle = {
	margin: '24px 0 0 0',
	fontSize: '13px',
	lineHeight: '18px',
	color: colors.fgMuted
};

const mutedLinkStyle = {
	color: colors.fgMuted,
	textDecoration: 'underline',
	wordBreak: 'break-word' as const
};
