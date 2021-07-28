import {Create, required, SimpleForm, TextInput, useGetIdentity} from "react-admin";
import React from "react";

const UserIdField = (({record}) => {
    const {identity, loading: identityLoading} = useGetIdentity();
    console.log(record, identity, identityLoading)

    return identityLoading ? null : <TextInput source="userName" defaultValue={identity.id} disabled={true}/>;
})

const CustomTransformer = (data) => {
    const {identity, loading: identityLoading} = useGetIdentity();

    data.addSomething = "lalala";
    return data;
}

const PlayerCreate = (props) => {
    const transform = data => ({
        ...data,
        fullName: `${data.firstName} ${data.lastName}`
    });
    return (<Create {...props} transform={CustomTransformer}>
        <SimpleForm variant={'outlined'}>
            <TextInput source="name" validate={[required()]}/>
            <UserIdField/>
        </SimpleForm>
    </Create>)
};

export default PlayerCreate
