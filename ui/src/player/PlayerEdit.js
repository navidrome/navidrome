import React from 'react'
import {
    BooleanInput,
    Button,
    Edit,
    ListButton,
    ReferenceField,
    ReferenceInput,
    RefreshButton,
    required,
    SelectInput,
    SimpleForm,
    TextField,
    TextInput,
    useTranslate, useUpdate,
} from 'react-admin'
import {Title} from '../common'
import config from '../config'
import CardActions from "@material-ui/core/CardActions";
import VpnKeyIcon from '@material-ui/icons/VpnKey';
import CloseIcon from '@material-ui/icons/Close';
import {v4 as uuidv4} from 'uuid'

const PostShowActions = ({basePath, data}) => (
    <CardActions>
        <ListButton label={'Back'} basePath={basePath}/>
        <RefreshButton/>
        <GenerateNewApiKey record={data}/>
        <RemoveApiKey record={data}/>
    </CardActions>
);

const PlayerTitle = ({record}) => {
    const translate = useTranslate()
    const resourceName = translate('resources.player.name', {smart_count: 1})
    return <Title subTitle={`${resourceName} ${record ? record.name : ''}`}/>
}

const HiddenApiUsernameField = ({record}) => {
    return record && record.apiKey ? <TextInput label="Api username" source='id' aria-readonly="true"/>  : null;
}

const HiddenApiField = ({record}) => {
    return record && record.apiKey ? <TextInput source='apiKey' aria-readonly="true"/> : null;
}

const GenerateNewApiKey = ({record}) => {
    const [update, { loading }] = useUpdate();
    const generateNewKey = () => update(
        'player',
        record.id,
        {...record, apiKey: uuidv4()}
    );

    return record && !record.apiKey ? <Button label="Generate new ApiKey" startIcon={<VpnKeyIcon/>} onClick={generateNewKey} disabled={loading}/> : null;
};

const RemoveApiKey = ({record}) => {
    const [update, { loading }] = useUpdate();
    const removeOldKey = () => update(
        'player',
        record.id,
        {...record, apiKey: ""}
    );

    return record && record.apiKey ? <Button label="Remove ApiKey" startIcon={<CloseIcon/>} onClick={removeOldKey} disabled={loading}/> : null;
};

const PlayerEdit = (props) => (
    <Edit title={<PlayerTitle/>} actions={<PostShowActions/>} {...props}>
        <SimpleForm variant={'outlined'}>
            <TextInput source="name" validate={[required()]}/>
            <ReferenceInput
                source="transcodingId"
                reference="transcoding"
                sort={{field: 'name', order: 'ASC'}}
            >
                <SelectInput source="name" resettable/>
            </ReferenceInput>
            <SelectInput
                source="maxBitRate"
                choices={[
                    {id: 32, name: '32'},
                    {id: 48, name: '48'},
                    {id: 64, name: '64'},
                    {id: 80, name: '80'},
                    {id: 96, name: '96'},
                    {id: 112, name: '112'},
                    {id: 128, name: '128'},
                    {id: 160, name: '160'},
                    {id: 192, name: '192'},
                    {id: 256, name: '256'},
                    {id: 320, name: '320'},
                    {id: 0, name: '-'},
                ]}
            />
            <BooleanInput source="reportRealPath" fullWidth/>
            {config.lastFMEnabled && (
                <BooleanInput source="scrobbleEnabled" fullWidth/>
            )}
            <TextField source="client"/>
            <ReferenceField label="User" source="userId" reference="user">
                <TextField source="name" />
            </ReferenceField>
            <HiddenApiUsernameField />
            <HiddenApiField />
        </SimpleForm>
    </Edit>
)

export default PlayerEdit
