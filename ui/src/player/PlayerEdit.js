import React from 'react'
import {
    BooleanInput,
    Button,
    Edit,
    ListButton,
    PasswordInput,
    ReferenceInput,
    RefreshButton,
    required,
    SelectInput,
    SimpleForm,
    TextField,
    TextInput,
    useMutation,
    useTranslate,
} from 'react-admin'
import {Title} from '../common'
import config from '../config'
import CardActions from "@material-ui/core/CardActions";
import VpnKeyIcon from '@material-ui/icons/VpnKey';
import CloseIcon from '@material-ui/icons/Close';
import {v4 as uuidv4} from 'uuid'

const PlayerTitle = ({record}) => {
    const translate = useTranslate()
    const resourceName = translate('resources.player.name', {smart_count: 1})
    return <Title subTitle={`${resourceName} ${record ? record.name : ''}`}/>
}

const GenerateNewApiKey = ({record}) => {
    const [mutate, { loading }] = useMutation();
    const generateNewKey = mutate({
        type: 'update',
        resource: 'player',
        payload: {
            id: record.id,
            data: {...record, apiKey: uuidv4()}
        }
    });
    return <Button label="Generate new ApiKey" startIcon={<VpnKeyIcon/>} onClick={generateNewKey} disabled={loading}/>;
};

const RemoveApiKey = ({record}) => {
    const [mutate, { loading }] = useMutation();
    const removeOldKey = mutate({
        type: 'update',
        resource: 'player',
        payload: {
            id: record.id,
            data: {...record, apiKey: ""}
        }
    });

    return <Button label="Remove ApiKey" startIcon={<CloseIcon/>} onClick={removeOldKey} disabled={loading}/>;
};

const PostShowActions = ({basePath, data}) => (
    <CardActions>
        <ListButton label={'Back'} basePath={basePath}/>
        <RefreshButton/>
        <GenerateNewApiKey record={data}/>
        <RemoveApiKey record={data}/>
    </CardActions>
);

const HiddenApiField = ({record}) => {
    return record && record.apiKey ? <PasswordInput source='apiKey' disabled={true}/> : null;
}

const PlayerEdit = (props) => (
    <Edit title={<PlayerTitle/>} actions={<PostShowActions/>} {...props}>
        <SimpleForm variant={'outlined'}>
            <TextInput source="name" validate={[required()]}/>
            <HiddenApiField/>
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
            <TextField source="userName"/>
        </SimpleForm>
    </Edit>
)

export default PlayerEdit
