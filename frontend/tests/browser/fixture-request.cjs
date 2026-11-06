const {TextDecoder}=require('node:util');
async function fixtureRequest(req) {
 if(!req.postData())return null;
 const headers=req.headers();
 if((headers['content-type']||'').includes('multipart/form-data')){
  const form=await new Response(req.postDataBuffer(),{headers:{'Content-Type':headers['content-type']}}).formData();
  const file=form.get('file'),bytes=file?Buffer.from(await file.arrayBuffer()):Buffer.alloc(0);let csv='',invalidUTF8=false;
  if(file?.name.endsWith('.csv'))try{csv=new TextDecoder('utf-8',{fatal:true}).decode(bytes)}catch{invalidUTF8=true}
  return {kind:form.get('kind'),sheet:form.get('sheet'),file:file?.name,fileBytes:bytes,csv,invalidUTF8};
 }
 return req.postDataJSON();
}
module.exports={fixtureRequest};
