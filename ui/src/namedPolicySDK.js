// Canonical shared authz SDK route until the generated client is refreshed.
// Authentication, namespace headers and stale-selection checks are supplied by
// StudioAPI.nativeRequest, exactly as for existing policy operations.
export async function postNamedPolicies(options) {
 const response = await options.fetch(`${options.baseUrl || ''}/v1/authz/sdk/namedPolicies.manage`, {
  method:'POST', credentials:options.credentials,
  headers:{...options.headers,'Content-Type':'application/json'}, body:JSON.stringify(options.body),
 });
 const payload=await response.json();
 return response.ok?{data:payload,response}:{error:payload,response};
}
