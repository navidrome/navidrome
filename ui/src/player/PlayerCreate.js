import {Create, required, SimpleForm, TextInput, useGetIdentity} from "react-admin";
import React from "react";

const UserIdField = (() => {
    const {identity, loading: identityLoading} = useGetIdentity();

    return identityLoading ? null : <TextInput source="userName" defaultValue={identity.id} disabled={true}/>;
})

const PlayerCreate = (props) => {
    return (<Create {...props}>
        <SimpleForm variant={'outlined'}>
            <TextInput source="name" validate={[required()]}/>
            <UserIdField/>
        </SimpleForm>
    </Create>)
};

export default PlayerCreate
