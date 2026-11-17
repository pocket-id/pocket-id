import { BaseTemplate } from '../components/base-template';
import CardHeader from '../components/card-header';
import { type SharedProps, sharedPreviewProps, sharedTemplateProps } from '../props';

export const TestEmail = (props: SharedProps) => (
	<BaseTemplate {...props} preview={props.t('test_description')}>
		<CardHeader title={props.t('test_title')}>{props.t('test_description')}</CardHeader>
	</BaseTemplate>
);

export default TestEmail;

TestEmail.TemplateProps = {
	...sharedTemplateProps
};

TestEmail.PreviewProps = {
	...sharedPreviewProps
};
