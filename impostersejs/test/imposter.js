function Get360Pan15 (request, state, logger) {
    var body = request.body;
    console.log("request body", body);
    var bodyJson = JSON.parse(body);
    var id = bodyJson.key_based_ids[0].id;
    console.log("ids", id);
    var xref = Math.random().toString(36).substr(2, 5);
    var response = "{\n" +
        "  \"Entities\":{\n" +
        "    \"individual_entities\": [\n" +
        "      {\n" +
        "        \"card_account_number15\": " + id + ",\n" +
        "        \"customer_identifiers\":\n" +
        "          {\n" +
        "            \"cust_xref_id\": " + xref + "\n" +
        "          }\n" +
        "      }\n" +
        "    ]\n" +
        "  }\n" +
        "}\n"
    return {
        statusCode : 200,
        headers: {
            'Content-Type': 'application/json; charset=utf-8'
        },
        body: response
    };
}