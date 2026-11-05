import { BaseTemplate } from '../components/base-template';
import CardHeader from '../components/card-header';
import { type SharedProps, sharedPreviewProps, sharedTemplateProps } from '../props';

export const TestEmail = (props: SharedProps) => (
	<BaseTemplate {...props} preview="Your email setup is working correctly">
		<CardHeader title="Test email">Your email setup is working correctly!</CardHeader>
	</BaseTemplate>
);

export default TestEmail;

TestEmail.TemplateProps = {
	...sharedTemplateProps
};

TestEmail.PreviewProps = {
	...sharedPreviewProps
};
