While developing the connector, please fill out this form. This information is needed to write docs and to help other users set up the connector.

## Connector capabilities

1. What resources does the connector sync?

    - Users
    - Roles
    - Groups


2. Can the connector provision any resources? If so, which ones? 
 
 Yes. The connector can:
- Create and delete users (agents).
- Manage group memberships by adding or removing users from groups.
- Manage role assignments by granting or revoking roles for users.

## Connector credentials 

1. What credentials or information are needed to set up the connector? (For example, API key, client ID and secret, domain, etc.)

- Freshdesk Domain
- API Key

2. For each item in the list above: 

   * How does a user create or look up that credential or info? Please include links to (non-gated) documentation, screenshots (of the UI or of gated docs), or a video of the process.

   **Freshdesk Domain:**
   This is the URL you use to access your Freshdesk account (e.g., for `your-company.freshdesk.com` ---> `your-company`).

   **API Key:**
   1. Log in to your Freshdesk account.
   2. Click on your profile picture in the top right corner and select **Profile Settings**.
   3. In the right pane, you will find your API Key. You may need to complete a CAPTCHA to view it.
   4. Copy this key.
   
   For more details, see [Freshdesk Support: How to find your API key](https://support.freshdesk.com/support/solutions/articles/215517-how-to-find-your-api-key).

   * Does the credential need any specific scopes or permissions? If so, list them here. 

   The API key is associated with a Freshdesk agent. The actions this connector can take are determined by the permissions of the agent whose API key is used. To use all the features of the connector (sync and provision), the agent should have an **Admin** or **Account Admin** role, which provides permissions to view and manage users, groups, and roles.

   * If applicable: Is the list of scopes or permissions different to sync (read) versus provision (read-write)? If so, list the difference here. 

   Yes, they are different:
   - **Sync (read-only):** Requires read permissions for Users (Agents), Groups, and Roles.
   - **Provision (read-write):** Requires permissions to create and delete Agents, update Groups (to manage members), and update Agents (to assign/unassign roles). An **Admin** role typically includes all necessary permissions for both sync and provisioning. 