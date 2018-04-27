layout(location=0) in vec3 in_v;
layout(location=2) in vec2 in_t;

uniform mat4 pm;     // projection matrix
uniform mat4 vm;     // view matrix
uniform mat4 mm;     // model matrix
uniform vec3  scale; // scale
out     vec2  v_t;   // pass uv coordinates through

void main() {
   mat4 mvm = vm * mm;
   gl_Position = pm * (vec4(in_v*scale, 1) + vec4(mvm[3].xyz, 0));
   v_t = in_t;
}
