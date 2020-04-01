## Creating New Themes

Themes in Navidrome are simple [Material-UI themes](https://material-ui.com/customization/theming/). They are basic JS 
objects, that allow you to override almost every visual aspect of Navidrome's UI.

#### Steps to create a new theme:

1) Create a new JS file in this folder that exports an object containing your theme. Create the theme based on the 
ReactAdmin/Material UI documentation below. See the existing themes for examples. 
2) Add a `themeName` property to your theme. This will be displayed in the theme selector
3) Add your new theme to the `ui/src/themes/index.js` file
4) Start the application, your new theme should now appear as an option in the theme selector

Before submitting a pull request to include your theme in Navidrome, please test your theme thoroughly and make sure 
it is formated with the [Prettier](https://prettier.io/) rules found in the project (`ui/src/.prettierrc.js`)

#### Resources for Material-UI theming

* Start reading [ReactAdmin documentation](https://marmelab.com/react-admin/Theming.html#writing-a-custom-theme)
* Color Tool: https://material-ui.com/customization/color/#official-color-tool  
