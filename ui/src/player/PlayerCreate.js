import {Create, ReferenceInput, required, SelectInput, SimpleForm, TextInput} from "react-admin";
import React from "react";

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

const PlayerCreate = ({permissions, ...props}) => {
    return (<Create {...props}>
        <SimpleForm variant={'outlined'}>
            <TextInput label="Player name" source="name" validate={[required()]}/>
            {permissions === 'admin' && <UserIdField />}
        </SimpleForm>
    </Create>)
};

export default PlayerCreate
