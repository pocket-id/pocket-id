import { DropdownMenu as DropdownMenuPrimitive } from 'bits-ui';
import Root from './dropdown-button.svelte';
import Main from './dropdown-button-main.svelte';
import Trigger from './dropdown-button-trigger.svelte';
import Content from './dropdown-button-content.svelte';
import Item from './dropdown-button-item.svelte';
import Separator from './dropdown-button-separator.svelte';

const DropdownRoot = DropdownMenuPrimitive.Root;
const DropdownTrigger = DropdownMenuPrimitive.Trigger;

export {
	Root,
	Main,
	Trigger,
	Content,
	Item,
	Separator,
	DropdownRoot,
	DropdownTrigger,
	//
	Root as DropdownButton,
	Main as DropdownButtonMain,
	Trigger as DropdownButtonTrigger,
	Content as DropdownButtonContent,
	Item as DropdownButtonItem,
	Separator as DropdownButtonSeparator,
	DropdownRoot as DropdownButtonRoot,
	DropdownTrigger as DropdownButtonPrimitiveTrigger
};
