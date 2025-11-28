# GX Codelab: online linear regression

This example illustrates how to run GX code, online linear regression
specifically, on an accelerator. The code is mainly composed of the following
files:

1.  [target.gx](target.gx):
    first generates the target to learn, that is a vector of weights. Samples
    can then be generated from such target.
2.  [learner.gx](learner.gx):
    learn from the samples generates by the target. The weights of the learner
    will converge to the weights of the target with an appropriate step size
    (a.k.a learning rate) parameter.
3.  [linearregression.go](linearregression.go):
    compile the GX code given a backend instance, then sequentially calls the
    target to generate a new sample and the learner to update the learned
    weights from that sample.
