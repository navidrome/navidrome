import {BooleanInput, Create, ReferenceInput, required, SelectInput, SimpleForm, TextInput} from "react-admin";
import React from "react";
import config from "../config";
import {v4 as uuidv4} from 'uuid'

const UserIdField = (() => {
    return (<ReferenceInput
        source="userId"
        reference="user"
        sort={{field: 'name', order: 'ASC'}}
        validate={[required()]}
    >
        <SelectInput source="userId" optionText="name" />
    </ReferenceInput>)
})

function PlayerCreateTransformer(data) {
    if (data.GenerateApiKey) {
        data.apiKey = uuidv4()
    }
    data.GenerateApiKey = undefined;
    return data;
}

const PlayerCreate = ({permissions, ...props}) => {
    return (<Create {...props} transform={PlayerCreateTransformer}>
        <SimpleForm variant={'outlined'}>
            <TextInput label="Player name" source="name" validate={[required()]}/>
            {permissions === 'admin' && <UserIdField />}
            <BooleanInput source="reportRealPath" fullWidth/>
            {config.lastFMEnabled && (
                <BooleanInput source="scrobbleEnabled" defaultValue={true} fullWidth/>
            )}
            <BooleanInput source="GenerateApiKey" fullWidth/>
        </SimpleForm>
    </Create>)
};

export default PlayerCreate
