import { Popover as PopoverPrimitive } from 'bits-ui';
import Content from './popover-content.svelte';
import Trigger from './popover-trigger.svelte';
const Root = PopoverPrimitive.Root;
const Close = PopoverPrimitive.Close;

export {
	Root,
	Content,
	Trigger,
	Close,
	//
	Root as Popover,
	Content as PopoverContent,
	Trigger as PopoverTrigger,
	Close as PopoverClose
};
